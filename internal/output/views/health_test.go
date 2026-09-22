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
				TargetDetails: &health.TargetDetails{},
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

		t.Run("formats blocker references", func(t *testing.T) {
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Target: []health.DependencyReport{{
				Scope:  health.DependencyScopeTarget,
				Name:   "Docker daemon",
				Status: health.CheckStatusUndetermined,
				BlockedBy: []health.DependencyBlocker{{
					Scope: health.DependencyScopeHost,
					Name:  "Docker CLI",
				}},
			}}}}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Deployment: undetermined (? 1)")
			assert.Contains(t, out.String(), " ? Docker daemon (not checked: requires host's Docker CLI)")
		})

		t.Run("gives errors precedence over undetermined checks", func(t *testing.T) {
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Host: []health.DependencyReport{
				{Scope: health.DependencyScopeHost, Name: "Docker CLI", Status: health.CheckStatusError},
			}, Target: []health.DependencyReport{
				{Scope: health.DependencyScopeTarget, Name: "Docker daemon", Status: health.CheckStatusUndetermined},
			}}}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Deployment: not ready (✗ 1 ? 1)")
		})
	})

	t.Run("AsPlain", func(t *testing.T) {
		t.Run("renders a warning-only report as ready", func(t *testing.T) {
			toPrint := views.HealthReport{
				ProjectDiscovery: health.ReadinessReport{
					TargetStatus: &health.TargetStatus{
						Status: health.CheckStatusWarning,
						Fix:    &health.Fix{Description: "provide --target"},
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Project management: ready (! 1)")
			assert.Contains(t, out.String(), "! Target\n   Fix:\n     provide --target")
		})
	})

	t.Run("AsJSON", func(t *testing.T) {
		t.Run("preserves the legacy combined target dependencies", func(t *testing.T) {
			toPrint := views.HealthReport{
				TargetDetails: &health.TargetDetails{Destination: "ssh://user@my-target"},
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

	t.Run("AsJSON", func(t *testing.T) {
		t.Run("formats blocker references in the value", func(t *testing.T) {
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Host: []health.DependencyReport{{
				Scope:  health.DependencyScopeHost,
				Name:   "Docker daemon",
				Status: health.CheckStatusUndetermined,
				BlockedBy: []health.DependencyBlocker{{
					Scope: health.DependencyScopeHost,
					Name:  "Docker CLI",
				}},
			}}}}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"host":{"dependencies":[{
					"name":"Docker daemon",
					"status":"undetermined",
					"value":"not checked: requires host's Docker CLI"
				}]}
			}`, out.String())
		})
	})
}
