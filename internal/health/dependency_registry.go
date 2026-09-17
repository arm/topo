package health

import (
	"context"
	"slices"
	"sync"
)

type DependencyNode struct {
	dependency    Dependency
	prerequisites []*DependencyNode
	once          sync.Once
	result        DependencyCheckResult
}

func (n *DependencyNode) Dependency() Dependency {
	return n.dependency
}

type DependencyRegistry struct {
	dependencies []*DependencyNode
}

func NewDependencyRegistry() *DependencyRegistry {
	return &DependencyRegistry{}
}

func (r *DependencyRegistry) Register(dependency Dependency, prerequisites ...*DependencyNode) *DependencyNode {
	for _, prerequisite := range prerequisites {
		r.assertRegistered(prerequisite)
	}

	node := &DependencyNode{dependency: dependency, prerequisites: prerequisites}
	r.dependencies = append(r.dependencies, node)
	return node
}

func (r *DependencyRegistry) Check(ctx context.Context, node *DependencyNode) (DependencyCheckResult, bool) {
	r.assertRegistered(node)
	for _, prerequisite := range node.prerequisites {
		prerequisiteResult, checked := r.Check(ctx, prerequisite)
		if !checked || prerequisiteResult.Failure != nil {
			return DependencyCheckResult{}, false
		}
	}
	return r.checkDependency(ctx, node), true
}

func (r *DependencyRegistry) checkDependency(ctx context.Context, node *DependencyNode) DependencyCheckResult {
	node.once.Do(func() {
		if node.dependency.Check != nil {
			node.result = node.dependency.Check(ctx)
		}
	})
	return node.result
}

func (r *DependencyRegistry) assertRegistered(node *DependencyNode) {
	if !slices.Contains(r.dependencies, node) {
		panic("health dependency is not registered in this registry")
	}
}
