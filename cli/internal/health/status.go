package health

import (
	"context"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type HardwareProfile struct {
	RemoteProcessors []probe.RemoteProcessor
	Err              error
}

func (h HardwareProfile) Capabilities() map[HardwareCapability]struct{} {
	capabilities := make(map[HardwareCapability]struct{})
	if len(h.RemoteProcessors) > 0 {
		capabilities[Remoteproc] = struct{}{}
	}
	return capabilities
}

type HealthStatus struct {
	Dependencies []DependencyStatus
	Hardware     HardwareProfile
}

func ProbeHealthStatus(ctx context.Context, r runner.Runner, target ssh.Destination) HealthStatus {
	var hs HealthStatus

	remoteProcessors, err := probe.Remoteproc(ctx, r)
	hs.Hardware.RemoteProcessors = remoteProcessors
	hs.Hardware.Err = err

	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies(target), hs.Hardware.Capabilities())
	hs.Dependencies = PerformChecks(ctx, dependenciesToCheck, r)

	return hs
}
