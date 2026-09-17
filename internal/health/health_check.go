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
	Host     []*DependencyNode
	Target   []*DependencyNode
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
	registry := NewDependencyRegistry()
	return HealthCheck{
		Registry: registry,
		Host:     hostRequiredDependencies(registry, options.SkipVersionChecks),
		Target:   targetRequiredDependencies(registry, options.Target, options.AcceptHostKeys, options.MissingTargetFixMessage),
	}
}

func (h HealthCheck) Evaluate(ctx context.Context) EvaluatedHealthCheck {
	return EvaluatedHealthCheck{
		Host:   h.evaluateDependencies(ctx, h.Host),
		Target: h.evaluateDependencies(ctx, h.Target),
	}
}

func (h HealthCheck) evaluateDependencies(ctx context.Context, dependencies []*DependencyNode) []EvaluatedDependency {
	statuses := make([]EvaluatedDependency, 0, len(dependencies))
	for _, reference := range dependencies {
		dependency := h.Registry.dependency(reference).dependency
		result, hasUnmetPrerequisites := h.Registry.Check(ctx, reference)
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
