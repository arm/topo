package health

import (
	"context"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type HealthStatus struct {
	Dependencies []DependencyStatus
}

func ProbeHealthStatus(ctx context.Context, r runner.Runner, target ssh.Destination, acceptNewHostKeys bool) HealthStatus {
	return HealthStatus{
		Dependencies: PerformChecks(ctx, TargetRequiredDependencies(target, acceptNewHostKeys), r),
	}
}
