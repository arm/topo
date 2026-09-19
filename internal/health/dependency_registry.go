package health

import (
	"context"
	"slices"
	"sync"
)

type DependencyNode struct {
	dependency   Dependency
	requirements DependencyRequirements
	once         sync.Once
	evaluation   DependencyEvaluation
}

type DependencyRequirements struct {
	Prerequisites []*DependencyNode
	Conditions    []*DependencyNode
}

type EvaluationState uint8

const (
	EvaluationExecuted EvaluationState = iota
	// EvaluationBlocked means an unsuccessful prerequisite prevented execution.
	EvaluationBlocked
	// EvaluationOmitted means an unsuccessful condition disabled the check.
	EvaluationOmitted
)

type DependencyEvaluation struct {
	State     EvaluationState
	Result    DependencyCheckResult
	BlockedBy []*DependencyNode
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

func (r *DependencyRegistry) Register(dependency Dependency, requirements DependencyRequirements) *DependencyNode {
	for _, condition := range requirements.Conditions {
		r.assertRegistered(condition)
	}
	for _, prerequisite := range requirements.Prerequisites {
		r.assertRegistered(prerequisite)
	}

	node := &DependencyNode{dependency: dependency, requirements: requirements}
	r.dependencies = append(r.dependencies, node)
	return node
}

func (r *DependencyRegistry) Check(ctx context.Context, node *DependencyNode) DependencyEvaluation {
	r.assertRegistered(node)
	node.once.Do(func() {
		node.evaluation = r.evaluate(ctx, node)
	})
	return node.evaluation
}

func (r *DependencyRegistry) evaluate(ctx context.Context, node *DependencyNode) DependencyEvaluation {
	for _, condition := range node.requirements.Conditions {
		if !r.succeeded(ctx, condition) {
			return DependencyEvaluation{State: EvaluationOmitted}
		}
	}

	blockedBy := make([]*DependencyNode, 0, len(node.requirements.Prerequisites))
	for _, prerequisite := range r.dependencies {
		isPrerequisite := slices.Contains(node.requirements.Prerequisites, prerequisite)
		if isPrerequisite && !r.succeeded(ctx, prerequisite) {
			blockedBy = append(blockedBy, prerequisite)
		}
	}
	if len(blockedBy) > 0 {
		return DependencyEvaluation{State: EvaluationBlocked, BlockedBy: blockedBy}
	}

	result := DependencyCheckResult{}
	if node.dependency.Check != nil {
		result = node.dependency.Check(ctx)
	}
	return DependencyEvaluation{State: EvaluationExecuted, Result: result}
}

func (r *DependencyRegistry) succeeded(ctx context.Context, node *DependencyNode) bool {
	evaluation := r.Check(ctx, node)
	return evaluation.State == EvaluationExecuted && evaluation.Result.Failure == nil
}

func (r *DependencyRegistry) assertRegistered(node *DependencyNode) {
	if !slices.Contains(r.dependencies, node) {
		panic("health dependency is not registered in this registry")
	}
}
