package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/deploy/podman/machine"
	"github.com/arm/topo/internal/runner"
)

const binfmtMiscPath = "/proc/sys/fs/binfmt_misc"

var ErrARM64BuildUnsupported = errors.New("linux/arm64 is not reported as supported")

func CheckPodmanARM64Build(ctx context.Context, r runner.CommandRunner) error {
	return checkPodmanARM64Build(ctx, r, os.DirFS(binfmtMiscPath))
}

func CheckDockerARM64Build(ctx context.Context, r runner.CommandRunner) error {
	output, _, err := r.Run(ctx, "docker buildx inspect")
	if err != nil {
		return err
	}
	var platforms []string
	for line := range strings.SplitSeq(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "Error":
			if value != "" {
				return fmt.Errorf("inspect Docker builder: %s", value)
			}
		case "Status":
			if value != "running" {
				return fmt.Errorf("selected Docker builder node is %s; start it before checking its platforms", value)
			}
		case "Platforms":
			if value != "" {
				platforms = append(platforms, strings.Split(value, ", ")...)
			}
		}
	}
	if len(platforms) == 0 {
		return errors.New("selected Docker builder capabilities are unavailable; inspect the builder with docker buildx inspect --bootstrap")
	}
	if slices.ContainsFunc(platforms, isARM64Platform) {
		return nil
	}
	return ErrARM64BuildUnsupported
}

func checkPodmanARM64Build(ctx context.Context, r runner.CommandRunner, binfmt fs.FS) error {
	output, _, err := r.Run(ctx, "podman info --format json")
	if err != nil {
		return err
	}
	var info struct {
		Host struct {
			Arch                  string
			OS                    string
			ServiceIsRemote       *bool
			EmulatedArchitectures []string
		}
	}
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return fmt.Errorf("parse Podman host capabilities: %w", err)
	}
	host := info.Host
	if host.Arch == "" || host.OS == "" {
		return errors.New("the Podman engine did not report its host architecture and operating system")
	}
	if host.OS != "linux" {
		return ErrARM64BuildUnsupported
	}
	if host.Arch == "arm64" || slices.ContainsFunc(host.EmulatedArchitectures, isARM64Platform) {
		return nil
	}
	if host.EmulatedArchitectures != nil {
		return ErrARM64BuildUnsupported
	}
	// Podman added emulatedArchitectures in 5.6; it is also omitted when empty.
	// Read local binfmt only for an explicitly local Linux engine.
	if host.ServiceIsRemote != nil && !*host.ServiceIsRemote {
		return checkARM64Binfmt(func(name string) ([]byte, error) {
			return fs.ReadFile(binfmt, name)
		})
	}
	return checkPodmanMachineARM64Build(ctx, r)
}

func checkPodmanMachineARM64Build(ctx context.Context, r runner.CommandRunner) error {
	connections, err := machine.Connections(ctx, r)
	if err != nil {
		return err
	}
	connectionIndex := slices.IndexFunc(connections, func(connection machine.Connection) bool {
		if name := os.Getenv("CONTAINER_CONNECTION"); name != "" {
			return connection.Name == name
		}
		if uri := os.Getenv("CONTAINER_HOST"); uri != "" {
			return connection.URI == uri
		}
		return connection.Default
	})
	if connectionIndex < 0 {
		return errors.New("no configured Podman connection matches the selected engine")
	}
	machines, err := machine.List(ctx, r)
	if err != nil {
		return err
	}
	machineName, err := machine.FindForConnection(connections[connectionIndex], machines)
	if err != nil {
		return err
	}
	return checkARM64Binfmt(func(name string) ([]byte, error) {
		output, _, err := r.Run(ctx, "podman machine ssh "+command.QuoteArg(machineName)+" cat "+binfmtMiscPath+"/"+name)
		return []byte(output), err
	})
}

func checkARM64Binfmt(readFile func(string) ([]byte, error)) error {
	status, err := readFile("status")
	if err != nil {
		return fmt.Errorf("read build host binfmt_misc status: %w", err)
	}
	switch strings.TrimSpace(string(status)) {
	case "disabled":
		return ErrARM64BuildUnsupported
	case "enabled":
	default:
		return errors.New("unrecognized build host binfmt_misc status")
	}
	// Standard registration installed by QEMU packages and tonistiigi/binfmt.
	registration, err := readFile("qemu-aarch64")
	if errors.Is(err, fs.ErrNotExist) {
		return ErrARM64BuildUnsupported
	}
	if err != nil {
		return fmt.Errorf("read ARM64 binfmt registration: %w", err)
	}
	state, fields, _ := strings.Cut(string(registration), "\n")
	if state != "enabled" {
		return ErrARM64BuildUnsupported
	}
	for line := range strings.SplitSeq(fields, "\n") {
		// F keeps the interpreter available across container mount namespaces.
		if flags, ok := strings.CutPrefix(line, "flags:"); ok && strings.Contains(flags, "F") {
			return nil
		}
	}
	return ErrARM64BuildUnsupported
}

func isARM64Platform(platform string) bool {
	platform = strings.TrimSuffix(platform, "*")
	return platform == "linux/arm64" || strings.HasPrefix(platform, "linux/arm64/")
}
