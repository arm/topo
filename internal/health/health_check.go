package health

import (
	"context"

	"github.com/arm/topo/internal/ssh"
)

type HealthCheckOptions struct {
	Target                  *ssh.Destination
	MissingTargetFixMessage string
	SkipVersionChecks       bool
	AcceptHostKeys          bool
}

type HealthCheck struct {
	Registry *DependencyRegistry
	Host     []DependencyID
	Target   []DependencyID
}

type EvaluatedDependency struct {
	ID     DependencyID
	Label  string
	Result DependencyCheckResult
}

type EvaluatedHealthCheck struct {
	Host   []EvaluatedDependency
	Target []EvaluatedDependency
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	hostDependencies := hostRequiredDependencies(options.SkipVersionChecks)
	targetDependencies := targetRequiredDependencies(options.Target, options.AcceptHostKeys, options.MissingTargetFixMessage)
	dependencies := append(hostDependencies, targetDependencies...)
	healthCheck := HealthCheck{
		Host:   dependencyIDs(hostDependencies),
		Target: dependencyIDs(targetDependencies),
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

func (h HealthCheck) evaluateDependencies(ctx context.Context, dependencies []DependencyID) []EvaluatedDependency {
	statuses := make([]EvaluatedDependency, 0, len(dependencies))
	for _, id := range dependencies {
		dependency := h.Registry.dependency(id).dependency
		result, hasUnmetPrerequisites := h.Registry.Check(ctx, id)
		if hasUnmetPrerequisites {
			continue
		}
		statuses = append(statuses, EvaluatedDependency{
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
