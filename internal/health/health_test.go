package health_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

func TestGenerateHostReport(t *testing.T) {
	t.Run("given two host dependencies in the same category, they are grouped in a health check", func(t *testing.T) {
		dependencyStatuses := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "foo", Category: "Baz"}, Error: nil},
			{Dependency: health.Dependency{Name: "bar", Category: "Baz"}, Error: nil},
		}

		got := health.GenerateHostReport(dependencyStatuses)

		want := health.HealthCheck{
			Name:   "Baz",
			Status: health.CheckStatusOK,
			Value:  "foo, bar",
		}
		assert.Contains(t, got.Dependencies, want)
	})

	t.Run("when a dependency is not installed, health check reports error", func(t *testing.T) {
		dependencyStatuses := []health.DependencyStatus{
			{
				Dependency: health.Dependency{Name: "whatever", Category: "Rube Golberg"},
				Error:      fmt.Errorf("whatever not found on path"),
			},
		}

		got := health.GenerateHostReport(dependencyStatuses)

		assert.Len(t, got.Dependencies, 1)
		assert.Equal(t, "Rube Golberg", got.Dependencies[0].Name)
		assert.Equal(t, health.CheckStatusError, got.Dependencies[0].Status)
		assert.Equal(t, "whatever not found on path", got.Dependencies[0].Value)
	})
}

func TestGenerateTargetReport(t *testing.T) {
	t.Run("when no remoteproc devices are found, SubsystemDriver health check reports error", func(t *testing.T) {
		ts := health.Status{}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusWarning, got.SubsystemDriver.Status)
		assert.Equal(t, "no remoteproc devices found", got.SubsystemDriver.Value)
	})

	t.Run("when remoteproc devices are found, SubsystemDriver status is ok and includes device names", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{
				RemoteCPU: []target.RemoteprocCPU{{Name: "m4_0"}, {Name: "m4_1"}},
			},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusOK, got.SubsystemDriver.Status)
		assert.Equal(t, "m4_0, m4_1", got.SubsystemDriver.Value)
	})

	t.Run("when no remoteproc devices are found, SubsystemDriver status reports a warning", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{RemoteCPU: nil},
		}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusWarning, got.SubsystemDriver.Status)
		assert.Equal(t, "no remoteproc devices found", got.SubsystemDriver.Value)
	})

	t.Run("when the target has a connection error, Connectivity status reports error", func(t *testing.T) {
		ts := health.Status{ConnectionError: assert.AnError}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusError, got.Connectivity.Status)
	})

	t.Run("when the target has no connection error, Connectivity status is ok", func(t *testing.T) {
		ts := health.Status{}

		got := health.GenerateTargetReport(ts)

		assert.Equal(t, health.CheckStatusOK, got.Connectivity.Status)
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

		got := health.GenerateTargetReport(ts)

		want := []health.HealthCheck{
			{Name: "bar", Status: health.CheckStatusOK, Value: "foo"},
		}
		assert.Equal(t, want, got.Dependencies)
	})
}
