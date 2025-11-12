package core_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/core"
	"github.com/arm-debug/topo-cli/internal/dependencies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractArmFeatures(t *testing.T) {
	t.Run("extracts mapped Arm features and ignores unrecognised", func(t *testing.T) {
		target := core.Target{
			Features: []string{"fp", "asimd", "sve2", "sme"},
		}
		res := core.ExtractArmFeatures(target)
		expected := []string{"NEON", "SVE2", "SME"}
		assert.Equal(t, expected, res)
	})

	t.Run("returns empty slice if no matching features", func(t *testing.T) {
		target := core.Target{Features: []string{"fp", "crc32"}}
		res := core.ExtractArmFeatures(target)
		assert.Empty(t, res)
	})
}

func TestGenerateReport(t *testing.T) {
	t.Run("given two host dependencies in the same category, they are grouped in a health check", func(t *testing.T) {
		dependencyStatuses := []dependencies.Status{
			{
				Dependency: dependencies.Dependency{Name: "foo", Category: "Baz"},
				Installed:  true,
			},
			{
				Dependency: dependencies.Dependency{Name: "bar", Category: "Baz"},
				Installed:  true,
			},
		}

		got := core.GenerateReport(dependencyStatuses, core.Target{})

		want := core.HealthCheck{
			Name:    "Baz",
			Healthy: true,
			Value:   "foo, bar",
		}
		assert.Contains(t, got.Host.Dependencies, want)
	})

	t.Run("when a dependency is not installed, the health check is unhealthy", func(t *testing.T) {
		dependencyStatuses := []dependencies.Status{
			{
				Dependency: dependencies.Dependency{Name: "whatever", Category: "Rube Golberg"},
				Installed:  false,
			},
		}

		got := core.GenerateReport(dependencyStatuses, core.Target{})

		assert.Len(t, got.Host.Dependencies, 1)
		assert.Equal(t, "Rube Golberg", got.Host.Dependencies[0].Name)
		assert.False(t, got.Host.Dependencies[0].Healthy)
	})

	t.Run("when the target has a connection error, Connectivity is unhealthy", func(t *testing.T) {
		unconnectedTarget := core.Target{ConnectionError: assert.AnError}

		got := core.GenerateReport(nil, unconnectedTarget)

		assert.False(t, got.Target.Connectivity.Healthy)
	})

	t.Run("when the target has no connection error, the Connectivity is healthy", func(t *testing.T) {
		connectedTarget := core.Target{}

		got := core.GenerateReport(nil, connectedTarget)

		assert.True(t, got.Target.Connectivity.Healthy)
	})

	t.Run("target features are listed", func(t *testing.T) {
		target := core.Target{
			ConnectionError: nil,
			Features:        []string{"asimd", "sve"},
		}

		got := core.GenerateReport(nil, target)

		assert.Equal(t, []string{"NEON", "SVE"}, got.Target.Features)
	})
}

func TestRenderReportAsPlainText(t *testing.T) {
	t.Run("it renders the dependencies", func(t *testing.T) {
		report := core.Report{}
		report.Host.Dependencies = []core.HealthCheck{{
			Name:    "Flux Capacitor",
			Healthy: true,
		}}

		got, err := core.RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.Contains(t, got, "Flux Capacitor")
	})

	t.Run("it renders connection failures", func(t *testing.T) {
		report := core.Report{}
		report.Target.Connectivity = core.HealthCheck{
			Name:    "Connected",
			Healthy: false,
		}

		got, err := core.RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.Contains(t, got, "Connected: ❌")
	})

	t.Run("when connected it renders cpu features", func(t *testing.T) {
		report := core.Report{}
		report.Target.Connectivity = core.HealthCheck{
			Name:    "Connected",
			Healthy: true,
		}
		report.Target.Features = []string{"FOO", "BAR"}

		got, err := core.RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.Contains(t, got, "FOO, BAR")
	})

	t.Run("when not connected, it does not render cpu features", func(t *testing.T) {
		report := core.Report{}
		report.Target.Connectivity = core.HealthCheck{
			Name:    "Connected",
			Healthy: false,
		}

		got, err := core.RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.NotContains(t, got, "Features")
	})
}
