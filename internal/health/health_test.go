package health_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateHostReport(t *testing.T) {
	testDependencyReporting(t, func(statuses []health.DependencyStatus) []health.HealthCheck {
		return health.GenerateHostReport(statuses).Dependencies
	})
}

func TestGenerateTargetReport(t *testing.T) {
	testDependencyReporting(t, func(statuses []health.DependencyStatus) []health.HealthCheck {
		return health.GenerateTargetReport(health.Status{Dependencies: statuses}).Dependencies
	})

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
}

func TestHostReport(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		t.Run("nil dependencies are [] not null", func(t *testing.T) {
			tr := health.HostReport{Dependencies: nil}

			b, err := json.Marshal(tr)

			require.NoError(t, err)
			want := `{ "dependencies": [] }`
			assert.JSONEq(t, want, string(b))
		})
	})
}

func TestTargetReport(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		t.Run("nil dependencies are [] not null", func(t *testing.T) {
			tr := health.TargetReport{Dependencies: nil}

			b, err := json.Marshal(tr)

			require.NoError(t, err)
			var result map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(b, &result))
			assert.JSONEq(t, `[]`, string(result["dependencies"]))
		})
	})
}

func testDependencyReporting(t *testing.T, extract func([]health.DependencyStatus) []health.HealthCheck) {
	t.Helper()

	t.Run("given two dependencies in the same category, they are grouped in a health check", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "foo", Category: "Baz"}, Error: nil},
			{Dependency: health.Dependency{Name: "bar", Category: "Baz"}, Error: nil},
		}

		got := extract(statuses)

		assert.Contains(t, got, health.HealthCheck{Name: "Baz", Status: health.CheckStatusOK, Value: "foo, bar"})
	})

	t.Run("when a dependency is not installed, health check reports error", func(t *testing.T) {
		statuses := []health.DependencyStatus{
			{Dependency: health.Dependency{Name: "whatever", Category: "Rube Goldberg"}, Error: fmt.Errorf("whatever not found on path")},
		}

		got := extract(statuses)

		assert.Equal(t, []health.HealthCheck{
			{Name: "Rube Goldberg", Status: health.CheckStatusError, Value: "whatever not found on path"},
		}, got)
	})
}
