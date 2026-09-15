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
		t.Run("evaluates a successful dependency once when shared", func(t *testing.T) {
			var evaluations atomic.Int32
			registry := health.NewDependencyRegistry([]health.Dependency{{
				ID: "shared",
				Check: func(context.Context) health.DependencyCheckResult {
					evaluations.Add(1)
					return health.DependencyCheckResult{SuccessValue: "ready"}
				},
			}})

			results := checkDependencyConcurrently(registry, "shared")

			assert.Equal(t, int32(1), evaluations.Load())
			assert.Equal(t, health.DependencyCheckResult{SuccessValue: "ready"}, results[0])
			assert.Equal(t, results[0], results[1])
		})

		t.Run("evaluates a failed dependency once when shared", func(t *testing.T) {
			var evaluations atomic.Int32
			failure := &health.DependencyCheckFailure{Message: "unavailable"}
			registry := health.NewDependencyRegistry([]health.Dependency{{
				ID: "shared",
				Check: func(context.Context) health.DependencyCheckResult {
					evaluations.Add(1)
					return health.DependencyCheckResult{Failure: failure}
				},
			}})

			results := checkDependencyConcurrently(registry, "shared")

			assert.Equal(t, int32(1), evaluations.Load())
			assert.Equal(t, health.DependencyCheckResult{Failure: failure}, results[0])
			assert.Equal(t, results[0], results[1])
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

func checkDependencyConcurrently(registry *health.DependencyRegistry, id health.DependencyID) [2]health.DependencyCheckResult {
	var results [2]health.DependencyCheckResult
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(results))

	for index := range results {
		go func() {
			defer waitGroup.Done()
			results[index] = registry.Check(context.Background(), id)
		}()
	}
	waitGroup.Wait()
	return results
}
