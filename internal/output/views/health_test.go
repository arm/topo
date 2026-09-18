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
	t.Run("AsPlain", func(t *testing.T) {
		t.Run("renders deployment and project management sections in verbose mode", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					TargetDetails: health.TargetDetails{},
					Deployment: health.ReadinessReport{
						Host: []health.DependencyReport{
							{Name: "Computer", Status: health.CheckStatusWarning},
							{Name: "Docker Compose", Status: health.CheckStatusError},
						},
						Target: []health.DependencyReport{
							{Name: "Docker API via SSH", Status: health.CheckStatusOK},
						},
					},
					ProjectDiscovery: health.ReadinessReport{
						Host: []health.DependencyReport{
							{Name: "OpenSSH", Status: health.CheckStatusOK},
						},
						Target: []health.DependencyReport{
							{Name: "Hardware Info (lscpu)", Status: health.CheckStatusOK},
						},
					},
				},
				Verbose: true,
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			want := `── Deployment: not ready (✗ 1 ! 1) ─────────────────────────
 ✗ Host
   ! Computer
   ✗ Docker Compose
 ✓ Target
   ✓ Docker API via SSH

── Project management: ready ───────────────────────────────
 ✓ Host
   ✓ OpenSSH
 ✓ Target
   ✓ Hardware Info (lscpu)
`

			require.NoError(t, err)
			assert.Equal(t, want, out.String())
		})

		t.Run("renders a warning-only report as ready", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					ProjectDiscovery: health.ReadinessReport{
						Target: []health.DependencyReport{{
							Name:   "Connectivity",
							Status: health.CheckStatusWarning,
							Value:  "target not specified; cannot calculate project compatibility",
						}},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Project management: ready (! 1)")
		})

		t.Run("it summarizes healthy host and target checks", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					TargetDetails: health.TargetDetails{Destination: "ssh://user@my-target"},
					Deployment: health.ReadinessReport{
						Host: []health.DependencyReport{
							{Name: "OpenSSH", Status: health.CheckStatusOK},
						},
						Target: []health.DependencyReport{
							{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK},
							{Name: "Container Engine", Status: health.CheckStatusOK},
							{ID: health.DependencyIDRemoteproc, Name: "Processing Domain Driver", Status: health.CheckStatusOK},
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ✓ Host\n   ✓ All checks passed\n")
			assert.Contains(t, out.String(), " ✓ Target: ssh://user@my-target\n   ✓ All checks passed\n")
		})

		t.Run("it renders the details when dependencies fail the health check", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					Deployment: health.ReadinessReport{
						Host: []health.DependencyReport{
							{Name: "OpenSSH", Status: health.CheckStatusOK},
							{
								Name:   "Container Engine",
								Status: health.CheckStatusError,
								Value:  "docker not found on path",
							},
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ✗ Host\n   ✗ Container Engine (docker not found on path)\n")
		})

		t.Run("it keeps informational checks alongside the success summary", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					Deployment: health.ReadinessReport{
						Target: []health.DependencyReport{
							{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK},
							{
								ID:     health.DependencyIDRemoteproc,
								Name:   "Processing Domain Driver (remoteproc)",
								Status: health.CheckStatusInfo,
								Value:  "no remoteproc devices found",
							},
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ✓ All checks passed")
			assert.Contains(t, out.String(), " i Processing Domain Driver (remoteproc) (no remoteproc devices found)")
		})

		t.Run("it renders the target destination when a dependency fails", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					TargetDetails: health.TargetDetails{Destination: "ssh://user@my-target"},
					Deployment: health.ReadinessReport{
						Target: []health.DependencyReport{
							{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK, Value: "ssh://user@my-target"},
							{Name: "Container Engine", Status: health.CheckStatusError},
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ✗ Target: ssh://user@my-target")
			assert.Contains(t, out.String(), " ✗ Container Engine")
		})

		t.Run("it renders the fix hint when a check has a fix", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					Deployment: health.ReadinessReport{
						Host: []health.DependencyReport{
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
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), " ! Skin Care")
			assert.Contains(t, out.String(), "     Fix:\n       Apply Working Hands Cream")
			assert.Contains(t, out.String(), "     Command:\n       topo moisturise")
		})

		t.Run("it colors status labels when writing to a terminal", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					Deployment: health.ReadinessReport{
						Host: []health.DependencyReport{
							{Name: "Healthy", Status: health.CheckStatusOK},
							{Name: "Broken", Status: health.CheckStatusError},
							{Name: "Deprecated", Status: health.CheckStatusWarning},
							{Name: "Skipped", Status: health.CheckStatusInfo},
						},
					},
				},
				Verbose: true,
			}

			out, err := toPrint.AsPlain(true)

			require.NoError(t, err)
			assert.Contains(t, out, term.Color(term.Dim, "── "))
			assert.Contains(t, out, term.Color(term.Green, " ✓ "))
			assert.Contains(t, out, term.Color(term.Red, " ✗ "))
			assert.Contains(t, out, term.Color(term.Yellow, " ! "))
			assert.Contains(t, out, term.Color(term.Blue, " i "))
		})
	})

	t.Run("AsJSON", func(t *testing.T) {
		t.Run("preserves the legacy combined target dependencies", func(t *testing.T) {
			toPrint := views.HealthReportView{
				HealthReport: health.HealthReport{
					TargetDetails: health.TargetDetails{Destination: "ssh://user@my-target"},
					Deployment: health.ReadinessReport{
						Host: []health.DependencyReport{{Name: "Topo", Status: health.CheckStatusOK}},
						Target: []health.DependencyReport{
							{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK},
							{Name: "Container Engine", Status: health.CheckStatusOK},
						},
					},
					ProjectDiscovery: health.ReadinessReport{
						Target: []health.DependencyReport{
							{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK},
							{Name: "Hardware Info", Status: health.CheckStatusOK},
						},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"host":{"dependencies":[{"name":"Topo","status":"ok","value":""}]},
				"target":{
					"destination":"ssh://user@my-target",
					"isLocalhost":false,
					"connectivity":{"name":"Connectivity","status":"ok","value":""},
					"dependencies":[
						{"name":"Container Engine","status":"ok","value":""},
						{"name":"Hardware Info","status":"ok","value":""}
					],
					"processingDomainDriver":{"name":"Processing Domain Driver (remoteproc)","status":"","value":""}
				}
			}`, out.String())
		})
	})
}
