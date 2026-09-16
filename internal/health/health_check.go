package health

import (
	"context"

	"github.com/arm/topo/internal/ssh"
)

type HealthCheckOptions struct {
	Target            *ssh.Destination
	SkipVersionChecks bool
	AcceptHostKeys    bool
}

type HealthCheck struct {
	Registry *DependencyRegistry
	Host     []DependencyID
	Target   []DependencyID
}

type DependencyStatus struct {
	ID     DependencyID
	Label  string
	Result DependencyCheckResult
}

type EvaluatedHealthCheck struct {
	Host   []DependencyStatus
	Target []DependencyStatus
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	hostDependencies := hostRequiredDependencies(options.SkipVersionChecks)
	dependencies := hostDependencies
	healthCheck := HealthCheck{
		Host: dependencyIDs(hostDependencies),
	}

	if options.Target != nil {
		targetDependencies := targetRequiredDependencies(*options.Target, options.AcceptHostKeys)
		dependencies = append(dependencies, targetDependencies...)
		healthCheck.Target = dependencyIDs(targetDependencies)
	}

	healthCheck.Registry = NewDependencyRegistry(dependencies)
	return healthCheck
}

func (h HealthCheck) Evaluate(ctx context.Context) EvaluatedHealthCheck {
	return EvaluatedHealthCheck{
		Host:   h.evaluateDependencies(ctx, h.Host),
		Target: h.evaluateDependencies(ctx, h.Target),
	}
}

func (h HealthCheck) evaluateDependencies(ctx context.Context, dependencies []DependencyID) []DependencyStatus {
	statuses := make([]DependencyStatus, 0, len(dependencies))
	for _, id := range dependencies {
		dependency := h.Registry.dependency(id).dependency
		result, hasUnmetPrerequisites := h.Registry.Check(ctx, id)
		if hasUnmetPrerequisites {
			continue
		}
		statuses = append(statuses, DependencyStatus{
			ID:     dependency.ID,
			Label:  dependency.Label,
			Result: result,
		})
	}
	return statuses
}

func dependencyIDs(dependencies []Dependency) []DependencyID {
	ids := make([]DependencyID, len(dependencies))
	for index, dependency := range dependencies {
		ids[index] = dependency.ID
	}
	return ids
}
