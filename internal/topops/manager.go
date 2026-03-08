package topops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/arm/topo/internal/output/console"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/target"
)

var containerIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{12,64}$`)

const (
	hostSubsystemName = "Host"
	remoteprocRuntime = "io.containerd.remoteproc.v1"
)

type Options struct {
	RootPath        string
	RefreshInterval time.Duration
}

type Manager struct {
	target string
	opts   Options
	log    *console.Logger
}

func NewManager(targetName string, opts Options, c *console.Logger) *Manager {
	return &Manager{
		target: targetName,
		opts:   opts,
		log:    c,
	}
}

type containerListItem struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
}

type inspectItem struct {
	ID         string `json:"Id"`
	HostConfig struct {
		Runtime     string            `json:"Runtime"`
		Annotations map[string]string `json:"Annotations"`
	} `json:"HostConfig"`
}

type containerInfo struct {
	ID          string
	Name        string
	Image       string
	State       string
	Status      string
	Runtime     string
	Annotations map[string]string
	Subsystem   string
}

func (m *Manager) Run(ctx context.Context) error {
	if err := os.MkdirAll(m.opts.RootPath, 0o750); err != nil {
		return fmt.Errorf("failed to create topology root %q: %w", m.opts.RootPath, err)
	}

	m.writeTopLevelReadme()

	if err := m.refresh(); err != nil {
		m.log.Log(logger.Entry{Level: logger.Warning, Message: err.Error()})
	}

	ticker := time.NewTicker(m.opts.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := m.processCommands(); err != nil {
				m.log.Log(logger.Entry{Level: logger.Warning, Message: err.Error()})
			}
			if err := m.refresh(); err != nil {
				m.log.Log(logger.Entry{Level: logger.Warning, Message: err.Error()})
			}
		}
	}
}

func (m *Manager) refresh() error {
	remoteprocs, containers, err := m.snapshot()
	if err != nil {
		return err
	}
	sort.Strings(remoteprocs)

	if err := m.syncFilesystem(remoteprocs, containers); err != nil {
		return err
	}
	return nil
}

func (m *Manager) snapshot() ([]string, []containerInfo, error) {
	conn := target.NewConnection(m.target, target.ConnectionOptions{
		Multiplex:      true,
		WithLoginShell: true,
	})

	remoteprocs, err := conn.ProbeRemoteproc()
	if err != nil {
		return nil, nil, fmt.Errorf("failed probing target processors: %w", err)
	}

	remoteprocNames := make([]string, 0, len(remoteprocs))
	for _, rp := range remoteprocs {
		remoteprocNames = append(remoteprocNames, rp.Name)
	}

	psOut, err := conn.Run(`docker ps --format "{{json .}}"`)
	if err != nil {
		return remoteprocNames, nil, fmt.Errorf("failed listing containers: %w", err)
	}

	items, err := parseContainerList(psOut)
	if err != nil {
		return remoteprocNames, nil, err
	}
	if len(items) == 0 {
		return remoteprocNames, nil, nil
	}

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	inspectOut, err := conn.Run(fmt.Sprintf("docker inspect %s --format '{{json .}}'", strings.Join(ids, " ")))
	if err != nil {
		return remoteprocNames, nil, fmt.Errorf("failed inspecting containers: %w", err)
	}

	inspectByID, err := parseInspect(inspectOut)
	if err != nil {
		return remoteprocNames, nil, err
	}

	containers := make([]containerInfo, 0, len(items))
	for _, item := range items {
		info := containerInfo{
			ID:          item.ID,
			Name:        item.Names,
			Image:       item.Image,
			State:       item.State,
			Status:      item.Status,
			Annotations: map[string]string{},
			Subsystem:   hostSubsystemName,
		}

		if inspect, ok := inspectByID[item.ID]; ok {
			info.Runtime = inspect.HostConfig.Runtime
			info.Annotations = inspect.HostConfig.Annotations
			if info.Runtime == remoteprocRuntime {
				if rpName := strings.TrimSpace(info.Annotations["remoteproc.name"]); rpName != "" {
					info.Subsystem = rpName
				}
			}
		}

		containers = append(containers, info)
	}

	return remoteprocNames, containers, nil
}

func parseContainerList(raw string) ([]containerListItem, error) {
	lines := splitNonEmptyLines(raw)
	items := make([]containerListItem, 0, len(lines))
	for _, line := range lines {
		var v containerListItem
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, fmt.Errorf("failed parsing docker ps output: %w", err)
		}
		items = append(items, v)
	}
	return items, nil
}

func parseInspect(raw string) (map[string]inspectItem, error) {
	lines := splitNonEmptyLines(raw)
	result := make(map[string]inspectItem, len(lines))
	for _, line := range lines {
		var v inspectItem
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, fmt.Errorf("failed parsing docker inspect output: %w", err)
		}
		id := strings.TrimSpace(v.ID)
		if id == "" {
			continue
		}
		shortID := id
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		result[shortID] = v
	}
	return result, nil
}

func splitNonEmptyLines(raw string) []string {
	out := []string{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func containerDirName(c containerInfo) string {
	name := sanitizePathComponent(c.Name)
	return fmt.Sprintf("%s--%s", name, c.ID)
}

func sanitizePathComponent(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "container"
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		string(os.PathSeparator), "_",
		":", "_",
		"\x00", "_",
	)
	v = replacer.Replace(v)
	v = strings.Join(strings.Fields(v), " ")
	if v == "" {
		return "container"
	}
	return v
}

func (m *Manager) syncFilesystem(remoteprocs []string, containers []containerInfo) error {
	root, err := filepath.Abs(m.opts.RootPath)
	if err != nil {
		return err
	}

	for _, subsystem := range append([]string{hostSubsystemName}, remoteprocs...) {
		subsystemPath := filepath.Join(root, sanitizePathComponent(subsystem))
		if err := os.MkdirAll(subsystemPath, 0o750); err != nil {
			return err
		}
	}

	expectedSubsystems := map[string]struct{}{}
	for _, subsystem := range append([]string{hostSubsystemName}, remoteprocs...) {
		expectedSubsystems[sanitizePathComponent(subsystem)] = struct{}{}
	}

	expectedContainerDirs := map[string]struct{}{}
	for _, c := range containers {
		subsystemPath := filepath.Join(root, sanitizePathComponent(c.Subsystem))
		cPath := filepath.Join(subsystemPath, containerDirName(c))
		expectedContainerDirs[cPath] = struct{}{}

		if err := os.MkdirAll(cPath, 0o750); err != nil {
			return err
		}
		if err := m.writeContainerFiles(cPath, c); err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subsystemPath := filepath.Join(root, entry.Name())
		if _, ok := expectedSubsystems[entry.Name()]; !ok {
			if err := os.RemoveAll(subsystemPath); err != nil {
				return err
			}
			continue
		}

		cEntries, err := os.ReadDir(subsystemPath)
		if err != nil {
			return err
		}
		for _, cEntry := range cEntries {
			if !cEntry.IsDir() {
				continue
			}
			cPath := filepath.Join(subsystemPath, cEntry.Name())
			if _, ok := expectedContainerDirs[cPath]; !ok {
				if err := os.RemoveAll(cPath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (m *Manager) writeContainerFiles(containerPath string, c containerInfo) error {
	type fileContent struct {
		name  string
		value string
	}
	files := []fileContent{
		{name: "id", value: c.ID + "\n"},
		{name: "name", value: c.Name + "\n"},
		{name: "state", value: c.State + "\n"},
		{name: "status", value: c.Status + "\n"},
		{name: "image", value: c.Image + "\n"},
		{name: "runtime", value: c.Runtime + "\n"},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(containerPath, f.name), []byte(f.value), 0o600); err != nil {
			return err
		}
	}

	commandFile := filepath.Join(containerPath, "command")
	if _, err := os.Stat(commandFile); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.WriteFile(commandFile, []byte{}, 0o600); err != nil {
			return err
		}
	}
	resultFile := filepath.Join(containerPath, "last_result")
	if _, err := os.Stat(resultFile); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.WriteFile(resultFile, []byte("pending\n"), 0o600); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) processCommands() error {
	root, err := filepath.Abs(m.opts.RootPath)
	if err != nil {
		return err
	}
	conn := target.NewConnection(m.target, target.ConnectionOptions{
		Multiplex:      true,
		WithLoginShell: true,
	})

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Name() != "command" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		command := strings.ToLower(strings.TrimSpace(string(raw)))
		if command == "" {
			return nil
		}

		containerIDPath := filepath.Join(filepath.Dir(path), "id")
		idRaw, err := os.ReadFile(containerIDPath)
		if err != nil {
			return err
		}
		containerID := strings.TrimSpace(string(idRaw))
		if containerID == "" {
			return fmt.Errorf("container id missing in %s", containerIDPath)
		}
		if !containerIDPattern.MatchString(containerID) {
			return fmt.Errorf("invalid container id %q in %s", containerID, containerIDPath)
		}

		var dockerCmd string
		switch command {
		case "start":
			dockerCmd = fmt.Sprintf("docker start %s", containerID)
		case "stop":
			dockerCmd = fmt.Sprintf("docker stop %s", containerID)
		default:
			m.log.Log(logger.Entry{
				Level:   logger.Warning,
				Message: fmt.Sprintf("unsupported command %q in %s; supported values are: start, stop", command, path),
			})
			if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(filepath.Dir(path), "last_result"), []byte("error: unsupported command\n"), 0o600)
		}

		_, err = conn.Run(dockerCmd)
		if err != nil {
			_ = os.WriteFile(filepath.Join(filepath.Dir(path), "last_result"), []byte(fmt.Sprintf("error: %v\n", err)), 0o600)
			if clearErr := os.WriteFile(path, []byte{}, 0o600); clearErr != nil {
				return clearErr
			}
			return nil
		}

		if err := os.WriteFile(filepath.Join(filepath.Dir(path), "last_result"), []byte("ok\n"), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
			return err
		}
		m.log.Log(logger.Entry{
			Level:   logger.Info,
			Message: fmt.Sprintf("container %s: %s", containerID, command),
		})
		return nil
	})
}

func (m *Manager) writeTopLevelReadme() {
	readmePath := filepath.Join(m.opts.RootPath, "README")
	content := strings.Join([]string{
		"Topo topology virtual filesystem",
		"",
		"Layout:",
		"  topology/Host/<container-name>--<id>/",
		"  topology/<processor>/<container-name>--<id>/",
		"    id, name, state, status, image, runtime",
		"    command      (write: start|stop)",
		"    last_result  (operation result)",
		"",
		"Example:",
		"  echo stop > topology/Host/my-service--abc123def456/command",
		"",
	}, "\n")
	_ = os.WriteFile(readmePath, []byte(content), 0o600)
}
