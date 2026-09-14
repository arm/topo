package health

import (
	"context"

	"github.com/arm/topo/internal/ssh"
)

type HealthStatus struct {
	Dependencies []DependencyStatus
}

func ProbeHealthStatus(ctx context.Context, target ssh.Destination, acceptNewHostKeys bool) HealthStatus {
	return HealthStatus{
		Dependencies: PerformChecks(ctx, TargetRequiredDependencies(target, acceptNewHostKeys)),
	}
}
