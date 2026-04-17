package health

import (
	"context"

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

	remoteprocs, _ := target.ProbeRemoteproc(ctx, r)
	hs.Hardware.RemoteCPU = remoteprocs

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, hs.Hardware.Capabilities())
	hs.Dependencies = PerformChecks(ctx, dependenciesToCheck, r)

	return hs
}
