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
			foreignRef := otherRegistry.Register(health.Dependency{Check: passingCheck})
			for name, ref := range map[string]*health.DependencyNode{
				"nil": nil, "zero": {}, "foreign": foreignRef,
			} {
				t.Run(name, func(t *testing.T) {
					assert.Panics(t, func() {
						registry.Check(context.Background(), ref)
					})
					assert.Panics(t, func() {
						registry.Register(health.Dependency{Prerequisites: []*health.DependencyNode{ref}})
					})
				})
			}
		})
	})

	t.Run("Check", func(t *testing.T) {
		t.Run("caches dependency evaluation", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			var evaluations atomic.Int32
			dependency := registry.Register(health.Dependency{Check: func(context.Context) health.DependencyCheckResult {
				evaluations.Add(1)
				return health.DependencyCheckResult{SuccessValue: "ready"}
			}})
			var results [2]health.DependencyCheckResult
			var waitGroup sync.WaitGroup

			waitGroup.Add(len(results))
			for index := range results {
				go func() {
					defer waitGroup.Done()
					results[index], _ = registry.Check(context.Background(), dependency)
				}()
			}
			waitGroup.Wait()

			assert.Equal(t, int32(1), evaluations.Load())
			assert.Equal(t, health.DependencyCheckResult{SuccessValue: "ready"}, results[0])
			assert.Equal(t, results[0], results[1])
		})

		t.Run("omits a dependency when a transitive prerequisite fails", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			flour := registry.Register(health.Dependency{ID: "flour", Check: failingCheck})
			dough := registry.Register(health.Dependency{
				ID:            "dough",
				Prerequisites: []*health.DependencyNode{flour},
				Check:         passingCheck,
			})
			evaluated := false
			pizza := registry.Register(health.Dependency{
				ID:            "pizza",
				Prerequisites: []*health.DependencyNode{dough},
				Check: func(context.Context) health.DependencyCheckResult {
					evaluated = true
					return health.DependencyCheckResult{SuccessValue: "pizza ready!"}
				},
			})

			_, hasUnmetPrerequisites := registry.Check(context.Background(), pizza)

			assert.True(t, hasUnmetPrerequisites)
			assert.False(t, evaluated)
		})

		t.Run("evaluates a dependency when a prerequisite passes", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			dough := registry.Register(health.Dependency{ID: "dough", Check: passingCheck})
			pizza := health.Dependency{
				ID:            "pizza",
				Prerequisites: []*health.DependencyNode{dough},
				Check: func(context.Context) health.DependencyCheckResult {
					return health.DependencyCheckResult{SuccessValue: "pizza ready!"}
				},
			}
			pizzaRef := registry.Register(pizza)

			got, hasUnmetPrerequisites := registry.Check(context.Background(), pizzaRef)

			assert.False(t, hasUnmetPrerequisites)
			wantResult := pizza.Check(context.Background())
			assert.Equal(t, wantResult, got)
		})
	})
}
