package health_test

import (
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/stretchr/testify/assert"
)

func TestGenerateReport(t *testing.T) {
	t.Run("given two host dependencies in the same category, they are grouped in a health check", func(t *testing.T) {
		dependencyStatuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "foo", Category: "Baz"},
				Installed:  true,
			},
			{
				Dependency: health.Dependency{Name: "bar", Category: "Baz"},
				Installed:  true,
			},
		}

		got := health.GenerateReport(dependencyStatuses, health.Status{})

		want := health.HealthCheck{
			Name:    "Baz",
			Healthy: true,
			Value:   "foo, bar",
		}
		assert.Contains(t, got.Host.Dependencies, want)
	})

	t.Run("when a dependency is not installed, the health check is unhealthy", func(t *testing.T) {
		dependencyStatuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "whatever", Category: "Rube Golberg"},
				Installed:  false,
			},
		}

		got := health.GenerateReport(dependencyStatuses, health.Status{})

		assert.Len(t, got.Host.Dependencies, 1)
		assert.Equal(t, "Rube Golberg", got.Host.Dependencies[0].Name)
		assert.False(t, got.Host.Dependencies[0].Healthy)
	})

	t.Run("when the target has a connection error, Connectivity is unhealthy", func(t *testing.T) {
		ts := health.Status{ConnectionError: assert.AnError}

		got := health.GenerateReport(nil, ts)

		assert.False(t, got.Target.Connectivity.Healthy)
	})

	t.Run("when the target has no connection error, the Connectivity is healthy", func(t *testing.T) {
		ts := health.Status{}

		got := health.GenerateReport(nil, ts)

		assert.True(t, got.Target.Connectivity.Healthy)
	})

	t.Run("target dependencies are listed", func(t *testing.T) {
		foo := health.Dependency{
			Name:     "foo",
			Category: "bar",
		}
		ts := health.Status{
			ConnectionError: nil,
			Dependencies: []health.DependencyStatus{
				{
					Dependency: foo,
					Installed:  true,
				},
			},
		}

		got := health.GenerateReport(nil, ts)

		want := []health.HealthCheck{
			{Name: "bar", Healthy: true, Value: "foo"},
		}
		assert.Equal(t, want, got.Target.Dependencies)
	})
}
