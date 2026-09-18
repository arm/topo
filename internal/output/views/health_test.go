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
		t.Run("renders deployment and project management sections", func(t *testing.T) {
			toPrint := views.HealthReport{
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
	})

	t.Run("AsPlain", func(t *testing.T) {
		t.Run("renders a warning-only report as ready", func(t *testing.T) {
			toPrint := views.HealthReport{
				ProjectDiscovery: health.ReadinessReport{
					Target: []health.DependencyReport{{
						Name:   "Connectivity",
						Status: health.CheckStatusWarning,
						Value:  "target not specified; cannot calculate project compatibility",
					}},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Project management: ready (! 1)")
		})
	})

	t.Run("AsJSON", func(t *testing.T) {
		t.Run("combines target dependencies including connectivity and processing domain drivers", func(t *testing.T) {
			toPrint := views.HealthReport{
				TargetDetails: health.TargetDetails{Destination: "ssh://user@my-target"},
				Deployment: health.ReadinessReport{
					Host: []health.DependencyReport{{Name: "Topo", Status: health.CheckStatusOK}},
					Target: []health.DependencyReport{
						{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK, Value: "ssh://user@my-target"},
						{Name: "Container Engine", Status: health.CheckStatusOK},
						{Name: "Processing Domain Driver (remoteproc)", Status: health.CheckStatusOK, Value: "remoteproc0"},
					},
				},
				ProjectDiscovery: health.ReadinessReport{
					Target: []health.DependencyReport{
						{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK, Value: "ssh://user@my-target"},
						{Name: "Hardware Info", Status: health.CheckStatusOK},
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
					"dependencies":[
						{"name":"Connectivity","status":"ok","value":"ssh://user@my-target"},
						{"name":"Container Engine","status":"ok","value":""},
						{"name":"Processing Domain Driver (remoteproc)","status":"ok","value":"remoteproc0"},
						{"name":"Hardware Info","status":"ok","value":""}
					]
				}
			}`, out.String())
		})

		t.Run("omits connectivity for plain localhost", func(t *testing.T) {
			toPrint := views.HealthReport{
				TargetDetails: health.TargetDetails{Destination: "localhost", IsLocalhost: true},
				Deployment: health.ReadinessReport{
					Target: []health.DependencyReport{
						{Name: "Container Engine", Status: health.CheckStatusOK, Value: "docker"},
						{Name: "Processing Domain Driver (remoteproc)", Status: health.CheckStatusInfo, Value: "no remoteproc devices found"},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"host":{"dependencies":[]},
				"target":{
					"destination":"localhost",
					"isLocalhost":true,
					"dependencies":[
						{"name":"Container Engine","status":"ok","value":"docker"},
						{"name":"Processing Domain Driver (remoteproc)","status":"info","value":"no remoteproc devices found"}
					]
				}
			}`, out.String())
		})

		t.Run("omits all other target dependencies when connectivity fails", func(t *testing.T) {
			toPrint := views.HealthReport{
				TargetDetails: health.TargetDetails{Destination: "ssh://user@my-target"},
				Deployment: health.ReadinessReport{
					Target: []health.DependencyReport{{ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusError, Value: "connection refused"}},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"host":{"dependencies":[]},
				"target":{
					"destination":"ssh://user@my-target",
					"isLocalhost":false,
					"dependencies":[{"name":"Connectivity","status":"error","value":"connection refused"}]
				}
			}`, out.String())
		})
	})
}
