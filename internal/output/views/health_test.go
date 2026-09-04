package views_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/output/views"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthReport(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("it renders the healthy host dependencies", func(t *testing.T) {
			toPrint := views.HealthReport{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:   "Flux Capacitor",
							Status: health.CheckStatusOK,
							Value:  "flux",
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "┌─ Host ")
			assert.Contains(t, out.String(), "Flux Capacitor")
			assert.Contains(t, out.String(), " ✓ Flux Capacitor (flux)")
		})

		t.Run("it renders the details when dependencies fail the health check", func(t *testing.T) {
			toPrint := views.HealthReport{
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

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Container Engine")
			assert.Contains(t, out.String(), " ✗ Container Engine (docker not found on path)")
		})

		t.Run("it renders a warning icon for warning checks", func(t *testing.T) {
			toPrint := views.HealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusOK,
					},
					ProcessingDomainDriver: health.HealthCheck{
						Name:   "Processing Domain Driver (remoteproc)",
						Status: health.CheckStatusWarning,
						Value:  "no remoteproc devices found",
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ! Processing Domain Driver (remoteproc) (no remoteproc devices found)")
		})

		t.Run("it renders an info icon for info checks", func(t *testing.T) {
			toPrint := views.HealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusOK,
					},
					ProcessingDomainDriver: health.HealthCheck{
						Name:   "Processing Domain Driver (remoteproc)",
						Status: health.CheckStatusInfo,
						Value:  "no remoteproc devices found",
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " i Processing Domain Driver (remoteproc) (no remoteproc devices found)")
		})

		t.Run("it renders connection failures", func(t *testing.T) {
			toPrint := views.HealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusError,
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ✗ Connected")
		})

		t.Run("it renders the target destination", func(t *testing.T) {
			toPrint := views.HealthReport{
				Target: &health.TargetReport{Destination: "ssh://user@my-target"},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "┌─ Target: ssh://user@my-target ")
		})

		t.Run("when not connected, it does not render cpu features", func(t *testing.T) {
			toPrint := views.HealthReport{
				Target: &health.TargetReport{
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusError,
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.NotContains(t, out.String(), "Features (Linux Host)")
		})

		t.Run("it renders the fix hint when a check has a fix", func(t *testing.T) {
			toPrint := views.HealthReport{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:   "Skin Care",
							Status: health.CheckStatusWarning,
							Fix: &health.Fix{
								Description: "Apply Working Hands Cream",
								Command:     "topo moisturise",
							},
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ! Skin Care")
			assert.Contains(t, out.String(), "   Fix:\n     Apply Working Hands Cream")
			assert.Contains(t, out.String(), "   Command:\n     topo moisturise")
		})

		t.Run("it colors status labels when writing to a terminal", func(t *testing.T) {
			toPrint := views.HealthReport{
				Host: health.HostReport{Dependencies: []health.HealthCheck{
					{Name: "Healthy", Status: health.CheckStatusOK},
					{Name: "Broken", Status: health.CheckStatusError},
					{Name: "Deprecated", Status: health.CheckStatusWarning},
					{Name: "Skipped", Status: health.CheckStatusInfo},
				}},
			}

			out, err := toPrint.AsPlain(true)

			require.NoError(t, err)
			assert.Contains(t, out, term.Color(term.Dim, "┌─ "))
			assert.Contains(t, out, term.Color(term.Green, " ✓ "))
			assert.Contains(t, out, term.Color(term.Red, " ✗ "))
			assert.Contains(t, out, term.Color(term.Yellow, " ! "))
			assert.Contains(t, out, term.Color(term.Blue, " i "))
		})

		t.Run("when no target is specified, prints the hint", func(t *testing.T) {
			hint := "Need to work on your aim"
			toPrint := views.HealthReport{TargetHint: hint}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "\n"+hint)
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
			toPrint := views.HealthReport{
				Host: health.HostReport{
					Dependencies: []health.HealthCheck{
						{
							Name:   "Flux Capacitor",
							Status: health.CheckStatusOK,
						},
					},
				},
				Target: &health.TargetReport{
					Destination: "ssh://user@my-target",
					Connectivity: health.HealthCheck{
						Name:   "Connected",
						Status: health.CheckStatusOK,
					},
					ProcessingDomainDriver: health.HealthCheck{
						Status: health.CheckStatusWarning,
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			want := `{
				"host": {
					"dependencies": [
						{"name":"Flux Capacitor","status":"ok","value":""}
					]
				},
				"target": {
					"destination": "ssh://user@my-target",
					"isLocalhost": false,
					"connectivity": {"name":"Connected","status":"ok","value":""},
					"dependencies": [],
					"processingDomainDriver": {"name":"","status":"warning","value":""}
				}
			}`
			assert.JSONEq(t, want, out.String())
		})
	})
}
