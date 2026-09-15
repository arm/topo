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

func (r *DependencyRegistry) Check(ctx context.Context, id DependencyID) DependencyCheckResult {
	dependency := r.dependency(id)

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

type DependencyGraphOptions struct {
	Target            *ssh.Destination
	SkipVersionChecks bool
	AcceptHostKeys    bool
}

type DependencyGraph struct {
	Registry *DependencyRegistry
	Host     []DependencyID
	Target   []DependencyID
}

type DependencyStatus struct {
	ID     DependencyID
	Label  string
	Result DependencyCheckResult
}

type EvaluatedDependencyGraph struct {
	Host   []DependencyStatus
	Target []DependencyStatus
}

func NewDependencyGraph(options DependencyGraphOptions) DependencyGraph {
	hostDependencies := hostRequiredDependencies(options.SkipVersionChecks)
	dependencies := hostDependencies
	graph := DependencyGraph{
		Host: dependencyIDs(hostDependencies),
	}

	if options.Target != nil {
		targetDependencies := targetRequiredDependencies(*options.Target, options.AcceptHostKeys)
		dependencies = append(dependencies, targetDependencies...)
		graph.Target = dependencyIDs(targetDependencies)
	}

	graph.Registry = NewDependencyRegistry(dependencies)
	return graph
}

func (g DependencyGraph) Evaluate(ctx context.Context) EvaluatedDependencyGraph {
	return EvaluatedDependencyGraph{
		Host:   g.evaluateDependencies(ctx, g.Host),
		Target: g.evaluateDependencies(ctx, g.Target),
	}
}

func (g DependencyGraph) evaluateDependencies(ctx context.Context, dependencies []DependencyID) []DependencyStatus {
	statuses := make([]DependencyStatus, 0, len(dependencies))
	for _, id := range dependencies {
		dependency := g.Registry.dependency(id).dependency
		if !prerequisitesFulfilled(ctx, g.Registry, dependency.Prerequisites) {
			continue
		}
		statuses = append(statuses, DependencyStatus{
			ID:     dependency.ID,
			Label:  dependency.Label,
			Result: g.Registry.Check(ctx, id),
		})
	}
	return statuses
}

func prerequisitesFulfilled(ctx context.Context, registry *DependencyRegistry, prerequisites []DependencyID) bool {
	for _, prerequisite := range prerequisites {
		dependency := registry.dependency(prerequisite).dependency
		if !prerequisitesFulfilled(ctx, registry, dependency.Prerequisites) {
			return false
		}
		if registry.Check(ctx, prerequisite).Failure != nil {
			return false
		}
	}
	return true
}

func dependencyIDs(dependencies []Dependency) []DependencyID {
	ids := make([]DependencyID, len(dependencies))
	for index, dependency := range dependencies {
		ids[index] = dependency.ID
	}
	return ids
}
