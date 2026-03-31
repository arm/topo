package health

import (
	"github.com/arm/topo/internal/command"
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

type HealthStatus struct {
	Dependencies []DependencyStatus
	Hardware     HardwareProfile
}

func ProbeHealthStatus(r runner) HealthStatus {
	var hs HealthStatus

	probe := target.NewHardwareProbe(r)
	remoteprocs, _ := probe.ProbeRemoteproc()
	hs.Hardware.RemoteCPU = remoteprocs

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, hs.Hardware.Capabilities())
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
	hs.Dependencies = PerformChecks(dependenciesToCheck, binaryExists, commandSuccessful)

	return hs
}
