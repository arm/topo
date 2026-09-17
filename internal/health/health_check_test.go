package health_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("evaluates deployment and project discovery checks", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			deployment := health.Dependency{ID: "deployment", Check: passingCheck}
			deploymentRef := registry.Register(deployment)
			projectDiscovery := health.Dependency{ID: "project-discovery", Check: failingCheck}
			projectDiscoveryRef := registry.Register(projectDiscovery)
			healthCheck := health.HealthCheck{
				Deployment:       health.ReadinessCheck{Registry: registry, Host: []*health.DependencyNode{deploymentRef}},
				ProjectDiscovery: health.ReadinessCheck{Registry: registry, Target: []*health.DependencyNode{projectDiscoveryRef}},
			}

			got := healthCheck.Evaluate(context.Background())

			want := health.EvaluatedHealthCheck{
				Deployment: health.EvaluatedReadinessCheck{
					Host: []health.EvaluatedDependency{{
						ID:     deployment.ID,
						Label:  deployment.Label,
						Result: deployment.Check(context.Background()),
					}},
					Target: []health.EvaluatedDependency{},
				},
				ProjectDiscovery: health.EvaluatedReadinessCheck{
					Host: []health.EvaluatedDependency{},
					Target: []health.EvaluatedDependency{{
						ID:     projectDiscovery.ID,
						Label:  projectDiscovery.Label,
						Result: projectDiscovery.Check(context.Background()),
					}},
				},
			}
			assert.Equal(t, want, got)
		})
	})
}

func TestReadinessCheck(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("reports successful dependencies in the group", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			virus := health.Dependency{ID: "virus", Check: passingCheck}
			virusRef := registry.Register(virus)
			bartek := health.Dependency{ID: "bartek", Check: passingCheck}
			bartekRef := registry.Register(bartek, virusRef)

			healthCheck := health.ReadinessCheck{
				Registry: registry,
				Host:     []*health.DependencyNode{bartekRef, virusRef},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{ID: bartek.ID, Label: bartek.Label, Result: bartek.Check(context.Background())},
				{ID: virus.ID, Label: virus.Label, Result: virus.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})

		t.Run("reports a failed prerequisite and omits its dependent", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			flour := health.Dependency{ID: "flour", Check: failingCheck}
			flourRef := registry.Register(flour)
			pizza := health.Dependency{ID: "pizza", Check: passingCheck}
			pizzaRef := registry.Register(pizza, flourRef)

			healthCheck := health.ReadinessCheck{
				Registry: registry,
				Host:     []*health.DependencyNode{flourRef, pizzaRef},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{ID: flour.ID, Label: flour.Label, Result: flour.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})
	})
}
