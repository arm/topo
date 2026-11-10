package podman

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/arm/topo/internal/deploy/podman/machine"
	"github.com/arm/topo/internal/runner"
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
	machines, err := machine.List(ctx, runner.NewLocal())
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
	connections, err := machine.Connections(ctx, runner.NewLocal())
	if err != nil {
		return machine.Connection{}, err
	}
	for _, connection := range connections {
		if connection.Default {
			return connection, nil
		}
	}
	return machine.Connection{}, fmt.Errorf("no default Podman connection is configured")
}
