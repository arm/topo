package templates_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintHealthReport(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("it renders the host dependencies", func(t *testing.T) {
			report := health.Report{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:    "Flux Capacitor",
							Healthy: true,
						},
					},
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				templates.PrintableHealthReport(report),
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			assert.Contains(t, out.String(), "Flux Capacitor")
			assert.Contains(t, out.String(), "✅")
		})

		t.Run("it renders the error detail for unhealthy dependencies", func(t *testing.T) {
			report := health.Report{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:    "Container Engine",
							Healthy: false,
							Value:   "docker not found on path",
						},
					},
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				templates.PrintableHealthReport(report),
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			assert.Contains(t, out.String(), "Container Engine")
			assert.Contains(t, out.String(), "❌")
			assert.Contains(t, out.String(), "docker not found on path")
		})

		t.Run("it renders connection failures", func(t *testing.T) {
			report := health.Report{
				Target: health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:    "Connected",
						Healthy: false,
					},
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				templates.PrintableHealthReport(report),
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			assert.Contains(t, out.String(), "Connected")
			assert.Contains(t, out.String(), "❌")
		})

		t.Run("when not connected, it does not render cpu features", func(t *testing.T) {
			report := health.Report{
				Target: health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:    "Connected",
						Healthy: false,
					},
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				templates.PrintableHealthReport(report),
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			assert.NotContains(t, out.String(), "Features (Linux Host)")
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
			report := health.Report{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:    "Flux Capacitor",
							Healthy: true,
						},
					},
				},
				Target: health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:    "Connected",
						Healthy: true,
					},
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				templates.PrintableHealthReport(report),
				&out,
				term.JSON,
			)
			require.NoError(t, err)

			want := `{
				"host": {
					"dependencies": [
						{"name":"Flux Capacitor","healthy":true,"value":""}
					]
				},
				"target": {
					"isLocalhost": false,
					"connectivity": {"name":"Connected","healthy":true,"value":""},
					"dependencies": [],
					"subsystemDriver": {"name":"","healthy":false,"value":""}
				}
			}`

			assert.JSONEq(t, want, out.String())
		})
	})
}
