package health

import (
	"context"
	"fmt"
	"sync"
)

type DependencyNode struct {
	dependency Dependency
	once       sync.Once
	result     DependencyCheckResult
}

type DependencyRegistry struct {
	dependencies map[DependencyID]*DependencyNode
}

func NewDependencyRegistry(dependencies []Dependency) *DependencyRegistry {
	registered := make(map[DependencyID]*DependencyNode, len(dependencies))
	for _, dependency := range dependencies {
		if _, exists := registered[dependency.ID]; exists {
			panic(fmt.Sprintf("duplicate health dependency ID: %q", dependency.ID))
		}
		registered[dependency.ID] = &DependencyNode{dependency: dependency}
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

func (r *DependencyRegistry) checkDependency(ctx context.Context, dependency *DependencyNode) DependencyCheckResult {
	dependency.once.Do(func() {
		if dependency.dependency.Check != nil {
			dependency.result = dependency.dependency.Check(ctx)
		}
	})
	return dependency.result
}

func (r *DependencyRegistry) dependency(id DependencyID) *DependencyNode {
	dependency, exists := r.dependencies[id]
	if !exists {
		panic(fmt.Sprintf("health dependency not registered: %q", id))
	}
	return dependency
}
