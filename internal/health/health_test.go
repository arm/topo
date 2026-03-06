package health_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

func TestGenerateReport(t *testing.T) {
	t.Run("given two host dependencies in the same category, they are grouped in a health check", func(t *testing.T) {
		dependencyStatuses := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "foo", Category: "Baz"}, Error: nil},
			{Dependency: health.Dependency{Name: "bar", Category: "Baz"}, Error: nil},
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
				Error:      fmt.Errorf("whatever not found on path"),
			},
		}

		got := health.GenerateReport(dependencyStatuses, health.Status{})

		assert.Len(t, got.Host.Dependencies, 1)
		assert.Equal(t, "Rube Golberg", got.Host.Dependencies[0].Name)
		assert.False(t, got.Host.Dependencies[0].Healthy)
		assert.Equal(t, "whatever not found on path", got.Host.Dependencies[0].Value)
	})

	t.Run("when no remoteproc devices are found, SubsystemDriver is unhealthy with an error message", func(t *testing.T) {
		ts := health.Status{}

		got := health.GenerateReport(nil, ts)

		assert.False(t, got.Target.SubsystemDriver.Healthy)
		assert.Equal(t, "no remoteproc devices found", got.Target.SubsystemDriver.Value)
	})

	t.Run("when remoteproc devices are found, SubsystemDriver is healthy with device names", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{
				RemoteCPU: []target.RemoteprocCPU{{Name: "m4_0"}, {Name: "m4_1"}},
			},
		}

		got := health.GenerateReport(nil, ts)

		assert.True(t, got.Target.SubsystemDriver.Healthy)
		assert.Equal(t, "m4_0, m4_1", got.Target.SubsystemDriver.Value)
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
					Error:      nil,
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
