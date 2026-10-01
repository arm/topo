package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/arm/topo/internal/deploy/podman/machine"
)

// Represents `.ConnectionInfo` JSON from `podman machine inspect`.
type podmanMachineConnectionInfo struct {
	PodmanSocket *machineSocket
	PodmanPipe   *machinePipe
}

type podmanMachineEndpoint struct {
	Name           string
	ConnectionInfo podmanMachineConnectionInfo
}

func resolvePodmanMachine(ctx context.Context, connection machine.Connection) (podmanMachineEndpoint, error) {
	machines, err := listPodmanMachines(ctx)
	if err != nil {
		return podmanMachineEndpoint{}, err
	}
	machineName, err := machine.FindForConnection(connection, machines)
	if err != nil {
		return podmanMachineEndpoint{}, err
	}
	info, err := inspectPodmanMachine(ctx, machineName)
	if err != nil {
		return podmanMachineEndpoint{}, fmt.Errorf("podman connection %q: %w", connection.Name, err)
	}
	return podmanMachineEndpoint{Name: machineName, ConnectionInfo: info}, nil
}

func listPodmanMachines(ctx context.Context) ([]machine.Identity, error) {
	output, err := exec.CommandContext(ctx, "podman", "machine", "list", "--format", "json").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Podman machines: %w", err)
	}
	var machines []machine.Identity
	if err := json.Unmarshal(output, &machines); err != nil {
		return nil, fmt.Errorf("failed to parse Podman machines: %w", err)
	}
	return machines, nil
}

func inspectPodmanMachine(ctx context.Context, machineName string) (podmanMachineConnectionInfo, error) {
	output, err := exec.CommandContext(ctx, "podman", "machine", "inspect", machineName, "--format", "{{json .ConnectionInfo}}").Output()
	if err != nil {
		return podmanMachineConnectionInfo{}, fmt.Errorf("failed to inspect Podman machine %q: %w", machineName, err)
	}
	var info podmanMachineConnectionInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return podmanMachineConnectionInfo{}, fmt.Errorf("failed to parse Podman machine %q connection information: %w", machineName, err)
	}
	return info, nil
}

func defaultPodmanConnection(ctx context.Context) (machine.Connection, error) {
	output, err := exec.CommandContext(ctx, "podman", "system", "connection", "list", "--format", "json").Output()
	if err != nil {
		return machine.Connection{}, fmt.Errorf("failed to list Podman connections: %w", err)
	}
	var connections []machine.Connection
	if err := json.Unmarshal(output, &connections); err != nil {
		return machine.Connection{}, fmt.Errorf("failed to parse Podman connections: %w", err)
	}
	for _, connection := range connections {
		if connection.Default {
			return connection, nil
		}
	}
	return machine.Connection{}, fmt.Errorf("no default Podman connection is configured")
}
