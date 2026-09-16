package health_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestDependencyRegistry(t *testing.T) {
	t.Run("Check", func(t *testing.T) {
		t.Run("evaluates each dependency once", func(t *testing.T) {
			var evaluations atomic.Int32
			keyboard := health.Dependency{
				ID: "keyboard",
				Check: func(context.Context) health.DependencyCheckResult {
					evaluations.Add(1)
					return health.DependencyCheckResult{SuccessValue: "ready"}
				},
			}
			registry := health.NewDependencyRegistry([]health.Dependency{keyboard})

			var results [2]health.DependencyCheckResult
			var waitGroup sync.WaitGroup
			waitGroup.Add(len(results))
			for index := range results {
				go func() {
					defer waitGroup.Done()
					results[index], _ = registry.Check(context.Background(), keyboard.ID)
				}()
			}
			waitGroup.Wait()

			assert.Equal(t, int32(1), evaluations.Load())
			assert.Equal(t, health.DependencyCheckResult{SuccessValue: "ready"}, results[0])
			assert.Equal(t, results[0], results[1])
		})

		t.Run("omits a dependency when a prerequisite fails", func(t *testing.T) {
			dough := health.Dependency{ID: "dough", Check: failingCheck}
			pizza := health.Dependency{
				ID:            "pizza",
				Prerequisites: []health.DependencyID{dough.ID},
				Check: func(context.Context) health.DependencyCheckResult {
					return health.DependencyCheckResult{SuccessValue: "pizza ready!"}
				},
			}
			registry := health.NewDependencyRegistry([]health.Dependency{pizza, dough})

			_, hasUnmetPrerequisites := registry.Check(context.Background(), pizza.ID)

			assert.True(t, hasUnmetPrerequisites)
		})

		t.Run("evaluates a dependency when a prerequisite passes", func(t *testing.T) {
			dough := health.Dependency{ID: "dough", Check: passingCheck}
			pizza := health.Dependency{
				ID:            "pizza",
				Prerequisites: []health.DependencyID{dough.ID},
				Check: func(context.Context) health.DependencyCheckResult {
					return health.DependencyCheckResult{SuccessValue: "pizza ready!"}
				},
			}
			registry := health.NewDependencyRegistry([]health.Dependency{pizza, dough})

			got, hasUnmetPrerequisites := registry.Check(context.Background(), pizza.ID)

			assert.False(t, hasUnmetPrerequisites)
			wantResult := pizza.Check(context.Background())
			assert.Equal(t, wantResult, got)
		})
	})
}

func TestNewDependencyGraph(t *testing.T) {
	t.Run("creates compatibility host and target groups", func(t *testing.T) {
		target := ssh.NewDestination("pi@edge-a")

		graph := health.NewDependencyGraph(health.DependencyGraphOptions{Target: &target})

		assert.NotNil(t, graph.Registry)
		assert.NotEmpty(t, graph.Host)
		assert.NotEmpty(t, graph.Target)
	})

	t.Run("creates only the host group without a target", func(t *testing.T) {
		graph := health.NewDependencyGraph(health.DependencyGraphOptions{})

		assert.NotNil(t, graph.Registry)
		assert.NotEmpty(t, graph.Host)
		assert.Empty(t, graph.Target)
	})
}

func TestDependencyGraph(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("reports successful dependencies in the group", func(t *testing.T) {
			virus := health.Dependency{ID: "virus", Check: passingCheck}
			bartek := health.Dependency{ID: "bartek", Prerequisites: []health.DependencyID{virus.ID}, Check: passingCheck}
			registry := health.NewDependencyRegistry([]health.Dependency{virus, bartek})

			graph := health.DependencyGraph{
				Registry: registry,
				Host:     []health.DependencyID{bartek.ID, virus.ID},
			}

			got := graph.Evaluate(context.Background())

			want := []health.DependencyStatus{
				{ID: bartek.ID, Label: bartek.Label, Result: bartek.Check(context.Background())},
				{ID: virus.ID, Label: virus.Label, Result: virus.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})

		t.Run("reports a failed prerequisite and omits its dependent", func(t *testing.T) {
			flour := health.Dependency{ID: "flour", Check: failingCheck}
			pizza := health.Dependency{ID: "pizza", Prerequisites: []health.DependencyID{flour.ID}, Check: passingCheck}
			registry := health.NewDependencyRegistry([]health.Dependency{flour, pizza})

			graph := health.DependencyGraph{
				Registry: registry,
				Host:     []health.DependencyID{flour.ID, pizza.ID},
			}

			got := graph.Evaluate(context.Background())

			want := []health.DependencyStatus{
				{ID: flour.ID, Label: flour.Label, Result: flour.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})
	})
}
