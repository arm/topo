package health_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestDependencyRegistry(t *testing.T) {
	t.Run("Register", func(t *testing.T) {
		t.Run("rejects invalid references", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			otherRegistry := health.NewDependencyRegistry()
			foreignRef := otherRegistry.Register(
				health.Dependency{Check: passingCheck},
				health.DependencyRequirements{},
			)

			for name, reference := range map[string]*health.DependencyNode{
				"nil":     nil,
				"zero":    {},
				"foreign": foreignRef,
			} {
				t.Run(name, func(t *testing.T) {
					assert.Panics(t, func() {
						registry.Check(context.Background(), reference)
					})
					assert.Panics(t, func() {
						registry.Register(
							health.Dependency{},
							health.DependencyRequirements{
								Prerequisites: []*health.DependencyNode{reference},
							},
						)
					})
					assert.Panics(t, func() {
						registry.Register(
							health.Dependency{},
							health.DependencyRequirements{
								Conditions: []*health.DependencyNode{reference},
							},
						)
					})
				})
			}
		})
	})

	t.Run("Check", func(t *testing.T) {
		t.Run("executes a dependency when requirements succeed", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			hungry := registerPassingDependency(registry)
			oven := registerPassingDependency(registry)
			pizza := health.Dependency{Check: func(context.Context) health.DependencyCheckResult {
				return health.DependencyCheckResult{SuccessValue: "pizza ready!"}
			}}
			pizzaRef := registry.Register(pizza, health.DependencyRequirements{
				Conditions:    []*health.DependencyNode{hungry},
				Prerequisites: []*health.DependencyNode{oven},
			})

			got := registry.Check(context.Background(), pizzaRef)

			want := health.DependencyEvaluation{
				State:  health.EvaluationExecuted,
				Result: pizza.Check(context.Background()),
			}
			assert.Equal(t, want, got)
		})

		t.Run("blocks a dependency when prerequisites are unsuccessful", func(t *testing.T) {
			for name, prerequisite := range map[string]dependencyRegistrar{
				"failed":  registerFailedDependency,
				"blocked": registerBlockedDependency,
				"omitted": registerOmittedDependency,
			} {
				t.Run(name, func(t *testing.T) {
					registry := health.NewDependencyRegistry()
					blocker := prerequisite(registry)
					evaluated := false
					subject := health.Dependency{
						Check: func(context.Context) health.DependencyCheckResult {
							evaluated = true
							return health.DependencyCheckResult{}
						},
					}
					subjectRef := registry.Register(subject, health.DependencyRequirements{
						Prerequisites: []*health.DependencyNode{blocker},
					})

					got := registry.Check(context.Background(), subjectRef)

					want := health.DependencyEvaluation{
						State:     health.EvaluationBlocked,
						BlockedBy: []*health.DependencyNode{blocker},
					}
					assert.Equal(t, want, got)
					assert.False(t, evaluated)
				})
			}
		})

		t.Run("omits a dependency when conditions are unsuccessful", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			failed := registerFailedDependency(registry)
			prerequisiteEvaluated := false
			prerequisite := registry.Register(
				health.Dependency{Check: func(context.Context) health.DependencyCheckResult {
					prerequisiteEvaluated = true
					return health.DependencyCheckResult{}
				}},
				health.DependencyRequirements{},
			)
			evaluated := false
			subject := registry.Register(
				health.Dependency{Check: func(context.Context) health.DependencyCheckResult {
					evaluated = true
					return health.DependencyCheckResult{}
				}},
				health.DependencyRequirements{
					Conditions:    []*health.DependencyNode{failed},
					Prerequisites: []*health.DependencyNode{prerequisite},
				})

			got := registry.Check(context.Background(), subject)

			assert.Equal(t, health.DependencyEvaluation{State: health.EvaluationOmitted}, got)
			assert.False(t, prerequisiteEvaluated)
			assert.False(t, evaluated)
		})

		t.Run("omits a dependency when a condition is blocked or omitted", func(t *testing.T) {
			for name, condition := range map[string]dependencyRegistrar{
				"blocked": registerBlockedDependency,
				"omitted": registerOmittedDependency,
			} {
				t.Run(name, func(t *testing.T) {
					registry := health.NewDependencyRegistry()
					condition := condition(registry)
					subject := health.Dependency{Check: passingCheck}
					subjectRef := registry.Register(subject, health.DependencyRequirements{
						Conditions: []*health.DependencyNode{condition},
					})

					got := registry.Check(context.Background(), subjectRef)

					assert.Equal(t, health.DependencyEvaluation{State: health.EvaluationOmitted}, got)
				})
			}
		})

		t.Run("retains transitive blocker causes", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			flour := registerFailedDependency(registry)
			dough := registry.Register(
				health.Dependency{Check: passingCheck},
				health.DependencyRequirements{Prerequisites: []*health.DependencyNode{flour}},
			)
			pizza := registry.Register(
				health.Dependency{Check: passingCheck},
				health.DependencyRequirements{Prerequisites: []*health.DependencyNode{dough}},
			)

			pizzaEval := registry.Check(context.Background(), pizza)
			wantPizzaEval := health.DependencyEvaluation{
				State:     health.EvaluationBlocked,
				BlockedBy: []*health.DependencyNode{dough},
			}
			assert.Equal(t, wantPizzaEval, pizzaEval)
			doughEval := registry.Check(context.Background(), dough)
			wantDoughEval := health.DependencyEvaluation{
				State:     health.EvaluationBlocked,
				BlockedBy: []*health.DependencyNode{flour},
			}
			assert.Equal(t, wantDoughEval, doughEval)
		})

		t.Run("retains immediate blockers in registration order", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			pineapple := registerFailedDependency(registry)
			cheese := registerFailedDependency(registry)
			pizza := registry.Register(
				health.Dependency{Check: passingCheck},
				health.DependencyRequirements{
					Prerequisites: []*health.DependencyNode{cheese, pineapple},
				},
			)

			got := registry.Check(context.Background(), pizza)

			want := health.DependencyEvaluation{
				State:     health.EvaluationBlocked,
				BlockedBy: []*health.DependencyNode{pineapple, cheese},
			}
			assert.Equal(t, want, got)
		})

		t.Run("caches complete evaluations during concurrent access", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			var evaluations atomic.Int32
			dependency := registry.Register(
				health.Dependency{Check: func(context.Context) health.DependencyCheckResult {
					evaluations.Add(1)
					return health.DependencyCheckResult{SuccessValue: "ready"}
				}},
				health.DependencyRequirements{},
			)
			var results [2]health.DependencyEvaluation
			var waitGroup sync.WaitGroup

			waitGroup.Add(len(results))
			for index := range results {
				go func() {
					defer waitGroup.Done()
					results[index] = registry.Check(context.Background(), dependency)
				}()
			}
			waitGroup.Wait()

			assert.Equal(t, int32(1), evaluations.Load())
			want := health.DependencyEvaluation{
				State:  health.EvaluationExecuted,
				Result: health.DependencyCheckResult{SuccessValue: "ready"},
			}
			assert.Equal(t, want, results[0])
			assert.Equal(t, results[0], results[1])
		})
	})
}

type dependencyRegistrar func(*health.DependencyRegistry) *health.DependencyNode

func registerPassingDependency(registry *health.DependencyRegistry) *health.DependencyNode {
	return registry.Register(
		health.Dependency{Check: passingCheck},
		health.DependencyRequirements{},
	)
}

func registerFailedDependency(registry *health.DependencyRegistry) *health.DependencyNode {
	return registry.Register(
		health.Dependency{Check: failingCheck},
		health.DependencyRequirements{},
	)
}

func registerBlockedDependency(registry *health.DependencyRegistry) *health.DependencyNode {
	failed := registerFailedDependency(registry)
	return registry.Register(
		health.Dependency{Check: passingCheck},
		health.DependencyRequirements{Prerequisites: []*health.DependencyNode{failed}},
	)
}

func registerOmittedDependency(registry *health.DependencyRegistry) *health.DependencyNode {
	failed := registerFailedDependency(registry)
	return registry.Register(
		health.Dependency{Check: passingCheck},
		health.DependencyRequirements{Conditions: []*health.DependencyNode{failed}},
	)
}
