package health

import (
	"context"
	"fmt"
	"sync"

	"github.com/arm/topo/internal/ssh"
)

type dependencyNode struct {
	dependency Dependency
	once       sync.Once
	result     DependencyCheckResult
}

type DependencyRegistry struct {
	dependencies map[DependencyID]*dependencyNode
}

func NewDependencyRegistry(dependencies []Dependency) *DependencyRegistry {
	registered := make(map[DependencyID]*dependencyNode, len(dependencies))
	for _, dependency := range dependencies {
		if _, exists := registered[dependency.ID]; exists {
			panic(fmt.Sprintf("duplicate health dependency ID: %q", dependency.ID))
		}
		registered[dependency.ID] = &dependencyNode{dependency: dependency}
	}
	return &DependencyRegistry{dependencies: registered}
}

func (r *DependencyRegistry) Check(ctx context.Context, id DependencyID) (DependencyCheckResult, bool) {
	dependency := r.dependency(id)
	for _, prerequisiteID := range dependency.dependency.Prerequisites {
		prerequisiteResult, hasUnmetPrerequisites := r.Check(ctx, prerequisiteID)
		if hasUnmetPrerequisites || prerequisiteResult.Failure != nil {
			return DependencyCheckResult{}, true
		}
	}
	return r.checkDependency(ctx, dependency), false
}

func (r *DependencyRegistry) checkDependency(ctx context.Context, dependency *dependencyNode) DependencyCheckResult {
	dependency.once.Do(func() {
		if dependency.dependency.Check != nil {
			dependency.result = dependency.dependency.Check(ctx)
		}
	})
	return dependency.result
}

func (r *DependencyRegistry) dependency(id DependencyID) *dependencyNode {
	dependency, exists := r.dependencies[id]
	if !exists {
		panic(fmt.Sprintf("health dependency not registered: %q", id))
	}
	return dependency
}

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
