package health

import (
	"context"
	"slices"
	"sync"
)

type DependencyNode struct {
	dependency Dependency
	once       sync.Once
	result     DependencyCheckResult
}

type DependencyRegistry struct {
	dependencies []*DependencyNode
}

func NewDependencyRegistry() *DependencyRegistry {
	return &DependencyRegistry{}
}

func (r *DependencyRegistry) Register(dependency Dependency) *DependencyNode {
	for _, prerequisite := range dependency.Prerequisites {
		r.dependency(prerequisite)
	}
	node := &DependencyNode{dependency: dependency}
	r.dependencies = append(r.dependencies, node)
	return node
}

func (r *DependencyRegistry) Check(ctx context.Context, node *DependencyNode) (DependencyCheckResult, bool) {
	r.dependency(node)
	for _, prerequisite := range node.dependency.Prerequisites {
		prerequisiteResult, hasUnmetPrerequisites := r.Check(ctx, prerequisite)
		if hasUnmetPrerequisites || prerequisiteResult.Failure != nil {
			return DependencyCheckResult{}, true
		}
	}
	return r.checkDependency(ctx, node), false
}

func (r *DependencyRegistry) checkDependency(ctx context.Context, node *DependencyNode) DependencyCheckResult {
	node.once.Do(func() {
		if node.dependency.Check != nil {
			node.result = node.dependency.Check(ctx)
		}
	})
	return node.result
}

func (r *DependencyRegistry) dependency(node *DependencyNode) *DependencyNode {
	if !slices.Contains(r.dependencies, node) {
		panic("health dependency is not registered in this registry")
	}
	return node
}
