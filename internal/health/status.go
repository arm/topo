package health

import (
	"context"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
)

type HardwareProfile struct {
	RemoteCPU []probe.RemoteprocCPU
	Err       error
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

	remoteprocs, err := probe.Remoteproc(ctx, r)
	hs.Hardware.RemoteCPU = remoteprocs
	hs.Hardware.Err = err

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, hs.Hardware.Capabilities())
	hs.Dependencies = PerformChecks(ctx, dependenciesToCheck, r)

	return hs
}
