package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Docker ps / inspect derived types
type DockerPsItem struct {
	Command      string `json:"Command"`
	CreatedAt    string `json:"CreatedAt"`
	ID           string `json:"ID"`
	Image        string `json:"Image"`
	Labels       string `json:"Labels"`
	LocalVolumes string `json:"LocalVolumes"`
	Mounts       string `json:"Mounts"`
	Names        string `json:"Names"`
	Networks     string `json:"Networks"`
	Ports        string `json:"Ports"`
	RunningFor   string `json:"RunningFor"`
	Size         string `json:"Size"`
	State        string `json:"State"`
	Status       string `json:"Status"`
}

type DockerPsItemWithRuntime struct {
	DockerPsItem
	Runtime string `json:"Runtime"`
	Ports   []int  `json:"HostPorts"`
}

// BuildComposeFile builds images for compose project.
func BuildComposeFile(composePath string) error {
	cmd := ExecCommand("docker", "--context", "default", "compose", "-f", composePath, "build")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build compose file: %s", stderr.String())
	}
	return nil
}

// FlashDockerFile saves image locally and loads on remote board context.
func FlashDockerFile(serviceName string) error {
	saveCmd := ExecCommand("docker", "--context", "default", "save", serviceName)
	sshCmd := ExecCommand("docker", "--context", DefaultDockerContext, "load")
	saveOut, err := saveCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe for docker save: %w", err)
	}
	sshCmd.Stdin = saveOut
	var saveStderr, sshStderr bytes.Buffer
	saveCmd.Stderr = &saveStderr
	sshCmd.Stderr = &sshStderr
	if err := saveCmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker save: %s", saveStderr.String())
	}
	if err := sshCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ssh docker-load: %s", sshStderr.String())
	}
	if err := saveCmd.Wait(); err != nil {
		return fmt.Errorf("docker save failed: %s", saveStderr.String())
	}
	if err := sshCmd.Wait(); err != nil {
		return fmt.Errorf("ssh docker-load failed: %s", sshStderr.String())
	}
	return nil
}

// ensureContextExists creates a docker context if absent.
func ensureContextExists(contextName, sshTarget string) error {
	exists, err := contextExists(contextName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return createContext(sshTarget, contextName)
}

func contextExists(contextName string) (bool, error) {
	cmd := ExecCommand("docker", "context", "ls", "--format", "{{.Name}}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return true, fmt.Errorf("failed to list docker contexts: %s", stderr.String())
	}
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		if scanner.Text() == contextName {
			return true, nil
		}
	}
	return false, nil
}

func createContext(sshTarget string, contextName string) error {
	host := fmt.Sprintf("ssh://%s", sshTarget)
	create := ExecCommand("docker", "context", "create", contextName, "--docker", fmt.Sprintf("host=%s", host))
	var stderr bytes.Buffer
	create.Stderr = &stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("failed to create docker context: %s", stderr.String())
	}
	return nil
}

// RunDockerComposeUp runs docker compose up -d --no-build with given context.
func RunDockerComposeUp(contextName, composePath string) error {
	cmd := ExecCommand("docker", "--context", contextName, "compose", "-f", composePath, "up", "-d", "--no-build")
	var out, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run docker compose up: %s", stderr.String())
	}
	return nil
}

// ReadContainersInfo returns enriched ps output.
func ReadContainersInfo(sshTarget string) ([]DockerPsItemWithRuntime, error) {
	dockerContext := getContextName(sshTarget)
	ensureContextExists(dockerContext, sshTarget)
	conn := []string{"--context", dockerContext}
	cmd := ExecCommand("docker", append(conn, "ps", "-a", "--format", "{{json .}}")...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := FilterNonEmpty(strings.Split(strings.TrimSpace(string(out)), "\n"))
	if len(lines) == 0 {
		return []DockerPsItemWithRuntime{}, nil
	}
	items := make([]DockerPsItem, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if !strings.HasPrefix(l, "{") {
			continue
		}
		var itm DockerPsItem
		_ = json.Unmarshal([]byte(l), &itm)
		items = append(items, itm)
	}
	ids := make([]string, len(items))
	for i, itm := range items {
		ids[i] = itm.ID
	}
	if len(ids) == 0 {
		return []DockerPsItemWithRuntime{}, nil
	}
	inspectArgs := append(append(conn, "inspect"), ids...)
	inspectArgs = append(inspectArgs, "--format", `{{json .NetworkSettings.Ports}};{{.HostConfig.Runtime}}`)
	inspectCmd := ExecCommand("docker", inspectArgs...)
	inspectOut, err := inspectCmd.Output()
	if err != nil {
		return nil, err
	}
	inspectLines := FilterNonEmpty(strings.Split(strings.TrimSpace(string(inspectOut)), "\n"))
	if len(inspectLines) != len(items) {
		return nil, fmt.Errorf("mismatch between ps items and inspect lines")
	}
	result := make([]DockerPsItemWithRuntime, len(items))
	for i, itm := range items {
		parts := strings.SplitN(inspectLines[i], ";", 2)
		var portsJSON, runtimeStr string
		if len(parts) >= 1 {
			portsJSON = parts[0]
		}
		if len(parts) == 2 {
			runtimeStr = parts[1]
		}
		hostPorts, _ := ParsePorts(portsJSON)
		if hostPorts == nil {
			hostPorts = []int{}
		}
		result[i] = DockerPsItemWithRuntime{DockerPsItem: itm, Runtime: runtimeStr, Ports: hostPorts}
	}
	return result, nil
}

// ParsePorts extracts host ports for container ports 80/443 from docker inspect port JSON.
func ParsePorts(portsJSON string) ([]int, error) {
	var portMap map[string][]struct {
		HostPort string `json:"HostPort"`
	}
	if err := json.Unmarshal([]byte(portsJSON), &portMap); err != nil {
		return nil, err
	}
	portSet := map[int]struct{}{}
	for key, mappings := range portMap {
		portStr := strings.Split(key, "/")[0]
		p, _ := strconv.Atoi(portStr)
		if (p == 80 || p == 443) && len(mappings) > 0 {
			for _, m := range mappings {
				if m.HostPort != "" {
					if hp, err := strconv.Atoi(m.HostPort); err == nil {
						portSet[hp] = struct{}{}
					}
				}
			}
		}
	}
	out := make([]int, 0, len(portSet))
	for hp := range portSet {
		out = append(out, hp)
	}
	return out, nil
}

// FilterNonEmpty removes blank lines.
func FilterNonEmpty(ss []string) []string {
	ret := make([]string, 0, len(ss))
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			ret = append(ret, t)
		}
	}
	return ret
}

// GetContainersInfo prints container info to stdout.
func GetContainersInfo(sshTarget string) error {
	items, err := ReadContainersInfo(sshTarget)
	if err != nil {
		return fmt.Errorf("failed to read containers info: %w", err)
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal containers info: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
