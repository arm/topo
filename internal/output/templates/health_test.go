package templates_test

import (
	"bytes"
	"fmt"
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
		t.Run("it renders the healthy host dependencies", func(t *testing.T) {
			toPrint := templates.PrintableHealthReport{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:   "Flux Capacitor",
							Status: health.CheckStatusOK,
						},
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Flux Capacitor")
			assert.Contains(t, out.String(), "✅")
		})

		t.Run("it renders the details when dependencies fail the health check", func(t *testing.T) {
			toPrint := templates.PrintableHealthReport{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:   "Container Engine",
							Status: health.CheckStatusError,
							Value:  "docker not found on path",
						},
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Container Engine")
			assert.Contains(t, out.String(), "❌")
			assert.Contains(t, out.String(), "docker not found on path")
		})

		t.Run("it renders a warning icon for warning checks", func(t *testing.T) {
			toPrint := templates.PrintableHealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusOK,
					},
					SubsystemDriver: health.HealthCheck{
						Name:   "Subsystem Driver (remoteproc)",
						Status: health.CheckStatusWarning,
						Value:  "no remoteproc devices found",
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "⚠️")
			assert.Contains(t, out.String(), "no remoteproc devices found")
		})

		t.Run("it renders connection failures", func(t *testing.T) {
			toPrint := templates.PrintableHealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusError,
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Connected")
			assert.Contains(t, out.String(), "❌")
		})

		t.Run("when not connected, it does not render cpu features", func(t *testing.T) {
			toPrint := templates.PrintableHealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusError,
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.NotContains(t, out.String(), "Features (Linux Host)")
		})

		t.Run("when no target is specified, prints the hint", func(t *testing.T) {
			hint := "Need to work on your aim"
			toPrint := templates.PrintableHealthReport{TargetHint: hint}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			want := fmt.Sprintf("ℹ️ %s", hint)
			assert.Contains(t, out.String(), want)
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
			toPrint := templates.PrintableHealthReport{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:   "Flux Capacitor",
							Status: health.CheckStatusOK,
						},
					},
				},
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusOK,
					},
					SubsystemDriver: health.HealthCheck{
						Status: health.CheckStatusWarning,
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			want := `{
				"host": {
					"dependencies": [
						{"name":"Flux Capacitor","status":"ok","value":""}
					]
				},
				"target": {
					"isLocalhost": false,
					"connectivity": {"name":"Connected","status":"ok","value":""},
					"dependencies": [],
					"subsystemDriver": {"name":"","status":"warning","value":""}
				}
			}`
			assert.JSONEq(t, want, out.String())
		})
	})
}
