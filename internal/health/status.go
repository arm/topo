package health

import (
	"context"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/target"
)

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

func ProbeHealthStatus(ctx context.Context, r runner.Runner) HealthStatus {
	var hs HealthStatus

	probe := target.NewHardwareProbe(r)
	remoteprocs, _ := probe.ProbeRemoteproc(ctx)
	hs.Hardware.RemoteCPU = remoteprocs

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, hs.Hardware.Capabilities())
	commandSuccessful := func(fullCmd string) error {
		_, err := r.Run(ctx, command.WrapInLoginShell(fullCmd))
		return err
	}
	hs.Dependencies = PerformChecks(ctx, dependenciesToCheck, r.BinaryExists, commandSuccessful)

	return hs
}
