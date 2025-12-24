package health_test

import (
	"bytes"
	"testing"

	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractArmFeatures(t *testing.T) {
	t.Run("extracts mapped Arm features and ignores unrecognised", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{
				Features: []string{"fp", "asimd", "sve2", "sme"},
			},
		}

		res := health.ExtractArmFeatures(ts)

		want := []string{"NEON", "SVE2", "SME"}
		assert.Equal(t, want, res)
	})

	t.Run("returns empty slice if no matching features", func(t *testing.T) {
		ts := health.Status{
			Hardware: health.HardwareProfile{
				Features: []string{"fp", "crc32"},
			},
		}

		res := health.ExtractArmFeatures(ts)

		assert.Empty(t, res)
	})
}

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

	t.Run("target features are listed", func(t *testing.T) {
		ts := health.Status{
			ConnectionError: nil,
			Hardware: health.HardwareProfile{
				Features: []string{"asimd", "sve"},
			},
		}

		got := health.GenerateReport(nil, ts)

		assert.Equal(t, []string{"NEON", "SVE"}, got.Target.Features)
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

func TestRenderReportAsPlainText(t *testing.T) {
	t.Run("it renders the dependencies", func(t *testing.T) {
		report := health.Report{}
		report.Host.Dependencies = []health.HealthCheck{{
			Name:    "Flux Capacitor",
			Healthy: true,
		}}
		var buf bytes.Buffer

		err := health.RenderReportAsPlainText(report, &buf)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Flux Capacitor")
	})

	t.Run("it renders connection failures", func(t *testing.T) {
		report := health.Report{}
		report.Target.Connectivity = health.HealthCheck{
			Name:    "Connected",
			Healthy: false,
		}
		var buf bytes.Buffer

		err := health.RenderReportAsPlainText(report, &buf)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Connected: ❌")
	})

	t.Run("when connected it renders cpu features", func(t *testing.T) {
		report := health.Report{}
		report.Target.Connectivity = health.HealthCheck{
			Name:    "Connected",
			Healthy: true,
		}
		report.Target.Features = []string{"FOO", "BAR"}
		var buf bytes.Buffer

		err := health.RenderReportAsPlainText(report, &buf)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "FOO, BAR")
	})

	t.Run("when not connected, it does not render cpu features", func(t *testing.T) {
		report := health.Report{}
		report.Target.Connectivity = health.HealthCheck{
			Name:    "Connected",
			Healthy: false,
		}
		var buf bytes.Buffer

		err := health.RenderReportAsPlainText(report, &buf)

		require.NoError(t, err)
		assert.NotContains(t, buf.String(), "Features")
	})
}

func TestRenderReportAsJSON(t *testing.T) {
	t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
		report := health.Report{
			Host: health.HostReport{
				Dependencies: []health.HealthCheck{
					{Name: "Flux Capacitor", Healthy: true},
				},
			},
			Target: health.TargetReport{
				Connectivity: health.HealthCheck{Name: "Connected", Healthy: true},
			},
		}
		var buf bytes.Buffer

		err := health.RenderReportAsJSON(report, &buf)
		require.NoError(t, err)

		want := `{
            "Host": {
              "Dependencies": [
                {"Name":"Flux Capacitor","Healthy":true,"Value":""}
              ]
            },
            "Target": {
              "Connectivity": {"Name":"Connected","Healthy":true,"Value":""},
              "Dependencies": [],
              "Features": [],
              "SubsystemDriver": {"Name":"","Healthy":false,"Value":""}
            }
          }`
		assert.JSONEq(t, want, buf.String())
	})
}
