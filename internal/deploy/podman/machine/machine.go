package machine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/arm/topo/internal/runner"
)

// Connection represents an entry from `podman system connection list`.
type Connection struct {
	Name         string
	URI          string
	IdentityPath string `json:"Identity"`
	Default      bool
	IsMachine    bool
}

// Identity represents the SSH identity of an entry from `podman machine list`.
type Identity struct {
	Name         string
	SSHPort      int `json:"Port"`
	IdentityPath string
}

func Connections(ctx context.Context, r runner.CommandRunner) ([]Connection, error) {
	output, _, err := r.Run(ctx, "podman system connection list --format json")
	if err != nil {
		return nil, fmt.Errorf("failed to list Podman connections: %w", err)
	}
	var connections []Connection
	if err := json.Unmarshal([]byte(output), &connections); err != nil {
		return nil, fmt.Errorf("failed to parse Podman connections: %w", err)
	}
	return connections, nil
}

func List(ctx context.Context, r runner.CommandRunner) ([]Identity, error) {
	output, _, err := r.Run(ctx, "podman machine list --format json")
	if err != nil {
		return nil, fmt.Errorf("failed to list Podman machines: %w", err)
	}
	var machines []Identity
	if err := json.Unmarshal([]byte(output), &machines); err != nil {
		return nil, fmt.Errorf("failed to parse Podman machines: %w", err)
	}
	return machines, nil
}

func FindForConnection(connection Connection, machines []Identity) (string, error) {
	sshPort, err := connectionSSHPort(connection)
	if err != nil {
		return "", err
	}
	for _, machine := range machines {
		if machine.SSHPort == sshPort && machine.IdentityPath == connection.IdentityPath {
			return machine.Name, nil
		}
	}
	return "", fmt.Errorf("podman connection %q: no Podman machine matches SSH port %d and identity %q", connection.Name, sshPort, connection.IdentityPath)
}

func connectionSSHPort(connection Connection) (int, error) {
	connectionURI, err := url.Parse(connection.URI)
	if err != nil {
		return 0, fmt.Errorf("podman connection %q has an invalid SSH URI: %w", connection.Name, err)
	}
	if connectionURI.Scheme != "ssh" {
		return 0, fmt.Errorf("podman connection %q requires an SSH URI", connection.Name)
	}
	port := connectionURI.Port()
	if port == "" {
		return 0, fmt.Errorf("podman connection %q requires an explicit SSH port", connection.Name)
	}
	sshPort, err := strconv.Atoi(port)
	if err != nil {
		return 0, fmt.Errorf("podman connection %q has an invalid SSH port %q: %w", connection.Name, port, err)
	}
	if connection.IdentityPath == "" {
		return 0, fmt.Errorf("podman connection %q has no SSH identity path", connection.Name)
	}
	return sshPort, nil
}
