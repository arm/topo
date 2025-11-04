package core

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/dependencies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractArmFeatures(t *testing.T) {
	t.Run("extracts mapped Arm features and ignores unrecognised", func(t *testing.T) {
		target := Target{
			features: []string{"fp", "asimd", "sve2", "sme"},
		}
		res := extractArmFeatures(target)
		expected := []string{"NEON", "SVE2", "SME"}
		assert.Equal(t, expected, res)
	})

	t.Run("returns empty slice if no matching features", func(t *testing.T) {
		target := Target{features: []string{"fp", "crc32"}}
		res := extractArmFeatures(target)
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

		got := GenerateReport(dependencyStatuses, Target{})

		want := HealthCheck{
			Name:    "Baz",
			Healthy: true,
			Value:   "foo, bar",
		}
		assert.Contains(t, got.HostDependencies, want)
	})

	t.Run("when a dependency is not installed, the health check is unhealthy", func(t *testing.T) {
		dependencyStatuses := []dependencies.Status{
			{
				Dependency: dependencies.Dependency{Name: "whatever", Category: "Rube Golberg"},
				Installed:  false,
			},
		}

		got := GenerateReport(dependencyStatuses, Target{})

		assert.Len(t, got.HostDependencies, 1)
		assert.Equal(t, "Rube Golberg", got.HostDependencies[0].Name)
		assert.False(t, got.HostDependencies[0].Healthy)
	})

	t.Run("when the target has a connection error, Connectivity is unhealthy", func(t *testing.T) {
		unconnectedTarget := Target{connectionError: assert.AnError}

		got := GenerateReport(nil, unconnectedTarget)

		assert.False(t, got.Connectivity.Healthy)
	})

	t.Run("when the target has no connection error, the Connectivity is healthy", func(t *testing.T) {
		connectedTarget := Target{}

		got := GenerateReport(nil, connectedTarget)

		assert.True(t, got.Connectivity.Healthy)
	})

	t.Run("target features are listed", func(t *testing.T) {
		target := Target{
			connectionError: nil,
			features:        []string{"asimd", "sve"},
		}

		got := GenerateReport(nil, target)

		assert.Equal(t, []string{"NEON", "SVE"}, got.TargetFeatures)
	})
}

func TestRenderReportAsPlainText(t *testing.T) {
	t.Run("it renders the dependencies", func(t *testing.T) {
		report := Report{
			HostDependencies: []HealthCheck{{
				Name:    "Flux Capacitor",
				Healthy: true,
				Value:   "",
			}},
		}

		got, err := RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.Contains(t, got, "Flux Capacitor")
	})

	t.Run("it renders connection failures", func(t *testing.T) {
		report := Report{
			Connectivity: HealthCheck{
				Name:    "Connected",
				Healthy: false,
				Value:   "",
			},
		}

		got, err := RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.Contains(t, got, "Connected: ❌")
	})

	t.Run("when connected it renders cpu features", func(t *testing.T) {
		report := Report{
			Connectivity: HealthCheck{
				Name:    "Connected",
				Healthy: true,
				Value:   "",
			},
			TargetFeatures: []string{"FOO", "BAR"},
		}

		got, err := RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.Contains(t, got, "FOO, BAR")
	})

	t.Run("when not connected, it does not renders cpu features", func(t *testing.T) {
		report := Report{
			Connectivity: HealthCheck{
				Name:    "Connected",
				Healthy: false,
				Value:   "",
			},
		}

		got, err := RenderReportAsPlainText(report)

		require.NoError(t, err)
		assert.NotContains(t, got, "Features")
	})
}
