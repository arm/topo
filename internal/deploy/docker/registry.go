package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/deploy/command"
)

const registryImage = "registry:2"

// EnsureRegistryRunning pulls the registry image and starts the existing local
// registry container, or creates it when it does not yet exist.
func EnsureRegistryRunning(ctx context.Context, output io.Writer, containerName, port string) error {
	if err := command.RunDocker(ctx, command.LocalHost, output, "pull", registryImage); err != nil {
		return err
	}

	if registryContainerExists(ctx, containerName) {
		if err := validateRegistryPort(ctx, containerName, port); err != nil {
			return err
		}
		return command.RunDocker(ctx, command.LocalHost, output, "start", containerName)
	}

	return runRegistryContainer(ctx, containerName, port, output)
}

func registryContainerExists(ctx context.Context, containerName string) bool {
	return command.RunDocker(ctx, command.LocalHost, io.Discard, "inspect", containerName) == nil
}

func validateRegistryPort(ctx context.Context, containerName, requestedPort string) error {
	cmd := command.DockerContext(ctx, command.LocalHost, "inspect", containerName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute %s: %w", strings.Join(cmd.Args, " "), err)
	}

	actualPort, err := registryHostPort(output)
	if err != nil {
		return fmt.Errorf("failed to inspect existing registry %s: %w", containerName, err)
	}
	if actualPort == requestedPort {
		return nil
	}
	return fmt.Errorf("registry port mismatch (running: %s, requested: %s)\nyou may need to stop the existing %s: docker rm -f %s", actualPort, requestedPort, containerName, containerName)
}

func registryHostPort(inspectOutput []byte) (string, error) {
	type portBinding struct {
		HostPort string `json:"HostPort"`
	}
	type containerInspect struct {
		HostConfig struct {
			PortBindings map[string][]portBinding `json:"PortBindings"`
		} `json:"HostConfig"`
	}

	var containers []containerInspect
	if err := json.Unmarshal(inspectOutput, &containers); err != nil {
		return "", fmt.Errorf("decode docker inspect output: %w", err)
	}
	if len(containers) != 1 {
		return "", fmt.Errorf("expected one inspected container, got %d", len(containers))
	}

	bindings := containers[0].HostConfig.PortBindings["5000/tcp"]
	if len(bindings) == 0 || bindings[0].HostPort == "" {
		return "", fmt.Errorf("container port 5000 is not published")
	}
	return bindings[0].HostPort, nil
}

func runRegistryContainer(ctx context.Context, containerName, port string, output io.Writer) error {
	var commandOutput bytes.Buffer
	combinedOutput := io.MultiWriter(output, &commandOutput)
	err := command.RunDocker(
		ctx,
		command.LocalHost,
		combinedOutput,
		"run",
		"-d",
		"--restart", "always",
		"-p", fmt.Sprintf("127.0.0.1:%s:5000", port),
		"--name", containerName,
		registryImage,
	)
	if err == nil {
		return nil
	}
	if strings.Contains(commandOutput.String(), "already in use") || strings.Contains(commandOutput.String(), "already allocated") {
		return fmt.Errorf("%w\nport is already in use, this could be an existing %s or another process", err, containerName)
	}
	return err
}
