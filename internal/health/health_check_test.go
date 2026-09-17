package health_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("reports successful dependencies in the group", func(t *testing.T) {
			virus := health.Dependency{ID: "virus", Check: passingCheck}
			bartek := health.Dependency{ID: "bartek", Prerequisites: []health.DependencyID{virus.ID}, Check: passingCheck}
			registry := health.NewDependencyRegistry([]health.Dependency{virus, bartek})

			healthCheck := health.HealthCheck{
				Registry: registry,
				Host:     []health.DependencyID{bartek.ID, virus.ID},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{ID: bartek.ID, Label: bartek.Label, Result: bartek.Check(context.Background())},
				{ID: virus.ID, Label: virus.Label, Result: virus.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})

		t.Run("reports a failed prerequisite and omits its dependent", func(t *testing.T) {
			flour := health.Dependency{ID: "flour", Check: failingCheck}
			pizza := health.Dependency{ID: "pizza", Prerequisites: []health.DependencyID{flour.ID}, Check: passingCheck}
			registry := health.NewDependencyRegistry([]health.Dependency{flour, pizza})

			healthCheck := health.HealthCheck{
				Registry: registry,
				Host:     []health.DependencyID{flour.ID, pizza.ID},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{ID: flour.ID, Label: flour.Label, Result: flour.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})
	})
}
