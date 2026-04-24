package health

import (
	"context"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
)

type HardwareProfile struct {
	RemoteCPUs []probe.RemoteprocCPU
	Err        error
}

func (h HardwareProfile) Capabilities() map[HardwareCapability]struct{} {
	capabilities := make(map[HardwareCapability]struct{})
	if len(h.RemoteCPUs) > 0 {
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
	hs.Hardware.RemoteCPUs = remoteprocs
	hs.Hardware.Err = err

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, hs.Hardware.Capabilities())
	hs.Dependencies = PerformChecks(ctx, dependenciesToCheck, r)

	return hs
}
