package health

import (
	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
)

type runner interface {
	Run(command string) (string, error)
}

type HardwareProfile struct {
	RemoteCPU []target.RemoteprocCPU
}

func (h HardwareProfile) Capabilities() map[HardwareCapability]struct{} {
	capabilities := make(map[HardwareCapability]struct{})
	if len(h.RemoteCPU) > 0 {
		capabilities[Remoteproc] = struct{}{}
	}
	return capabilities
}

type Status struct {
	SSHTarget       ssh.Destination
	ConnectionError error
	Dependencies    []DependencyStatus
	Hardware        HardwareProfile
}

func ProbeHealthStatus(r runner) Status {
	var status Status

	probe := target.NewHardwareProbe(r)
	remoteprocs, _ := probe.ProbeRemoteproc()
	status.Hardware.RemoteCPU = remoteprocs

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, status.Hardware.Capabilities())
	binaryExists := func(bin string) error {
		// We can use `UnsafeBinaryLookupCommand`, because the dependencies we're checking are hardcoded in the codebase
		if _, err := r.Run(command.UnsafeBinaryLookupCommand(bin)); err != nil {
			return err
		}
		return nil
	}
	commandSuccessful := func(fullCmd string) error {
		_, err := r.Run(command.WrapInLoginShell(fullCmd))
		return err
	}
	status.Dependencies = PerformChecks(dependenciesToCheck, binaryExists, commandSuccessful)

	return status
}
