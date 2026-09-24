package views_test

import (
	"bytes"
	"encoding/json"
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
				Deployment: health.ReadinessReport{Checks: []health.DependencyReport{
					{Scope: health.DependencyScopeHost, Name: "Computer", Status: health.CheckStatusWarning},
					{Scope: health.DependencyScopeHost, Name: "Docker Compose", Status: health.CheckStatusError},
					{Scope: health.DependencyScopeTarget, Name: "Docker API via SSH", Status: health.CheckStatusOK},
				}},
				ProjectDiscovery: health.ReadinessReport{Checks: []health.DependencyReport{
					{Scope: health.DependencyScopeHost, Name: "OpenSSH", Status: health.CheckStatusOK},
					{Scope: health.DependencyScopeTarget, Name: "Hardware Info (lscpu)", Status: health.CheckStatusOK},
				}},
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
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Checks: []health.DependencyReport{{
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
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Checks: []health.DependencyReport{
				{Scope: health.DependencyScopeHost, Name: "Docker CLI", Status: health.CheckStatusError},
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
		t.Run("omits capabilities without checks or issues", func(t *testing.T) {
			toPrint := views.HealthReport{}

			got, err := toPrint.AsJSON()

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"capabilities": []
			}`, got)
		})

		t.Run("retains an empty capability with a warning but no fix", func(t *testing.T) {
			toPrint := views.HealthReport{ProjectDiscovery: health.ReadinessReport{
				TargetStatus: &health.TargetStatus{Status: health.CheckStatusWarning},
			}}

			got, err := toPrint.AsJSON()

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"capabilities": [
					{
						"name": "Project management",
						"status": "warning",
						"checks": []
					}
				]
			}`, got)
		})

		t.Run("retains an empty capability with a fix", func(t *testing.T) {
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{
				TargetStatus: &health.TargetStatus{
					Status: health.CheckStatusOK,
					Fix:    &health.Fix{Description: "provide --target"},
				},
			}}

			got, err := toPrint.AsJSON()

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"capabilities": [
					{
						"name": "Deployment",
						"status": "ok",
						"fix": {
							"description": "provide --target"
						},
						"checks": []
					}
				]
			}`, got)
		})

		t.Run("reports missing target fixes on each capability", func(t *testing.T) {
			report := (health.EvaluatedHealthCheck{}).Report(nil, "provide --target or set TOPO_TARGET to check target health")
			toPrint := views.HealthReport{Deployment: report.Deployment, ProjectDiscovery: report.ProjectDiscovery}

			got, err := toPrint.AsJSON()

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"capabilities": [
					{
						"name": "Deployment",
						"status": "error",
						"fix": {
							"description": "provide --target or set TOPO_TARGET to check target health"
						},
						"checks": []
					},
					{
						"name": "Project management",
						"status": "warning",
						"fix": {
							"description": "provide --target or set TOPO_TARGET to check target health"
						},
						"checks": []
					}
				]
			}`, got)
		})

		t.Run("preserves failure messages and fixes", func(t *testing.T) {
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Checks: []health.DependencyReport{{
				Scope: health.DependencyScopeHost, Name: "Docker CLI", Status: health.CheckStatusError, Value: "docker not found",
				Fix: &health.Fix{Description: "Install Docker", Command: "install-docker"},
			}}}}

			got, err := toPrint.AsJSON()

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"capabilities": [
					{
						"name": "Deployment",
						"status": "error",
						"checks": [
							{
								"name": "Docker CLI",
								"location": "host",
								"status": "error",
								"value": "docker not found",
								"fix": {
									"description": "Install Docker",
									"command": "install-docker"
								}
							}
						]
					}
				]
			}`, got)
		})

		t.Run("aggregates capability status", func(t *testing.T) {
			for _, scenario := range []struct {
				name   string
				host   health.CheckStatus
				target health.CheckStatus
				want   health.CheckStatus
			}{
				{"errors override blocked checks", health.CheckStatusUndetermined, health.CheckStatusError, health.CheckStatusError},
				{"blocked checks override warnings", health.CheckStatusWarning, health.CheckStatusUndetermined, health.CheckStatusUndetermined},
				{"warnings remain visible", health.CheckStatusOK, health.CheckStatusWarning, health.CheckStatusWarning},
				{"information does not reduce readiness", health.CheckStatusOK, health.CheckStatusInfo, health.CheckStatusOK},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					toPrint := views.HealthReport{Deployment: health.ReadinessReport{Checks: []health.DependencyReport{
						{Scope: health.DependencyScopeHost, Status: scenario.host},
						{Scope: health.DependencyScopeTarget, Status: scenario.target},
					}}}

					got, err := toPrint.AsJSON()

					require.NoError(t, err)
					var report struct {
						Capabilities []struct{ Status health.CheckStatus }
					}
					require.NoError(t, json.Unmarshal([]byte(got), &report))
					require.Len(t, report.Capabilities, 1)
					assert.Equal(t, scenario.want, report.Capabilities[0].Status)
				})
			}
		})

		t.Run("groups host and target checks by capability", func(t *testing.T) {
			toPrint := views.HealthReport{
				Deployment: health.ReadinessReport{Checks: []health.DependencyReport{
					{Scope: health.DependencyScopeHost, Name: "Topo", Status: health.CheckStatusOK},
					{Scope: health.DependencyScopeTarget, ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK},
				}},
				ProjectDiscovery: health.ReadinessReport{Checks: []health.DependencyReport{
					{Scope: health.DependencyScopeTarget, ID: health.DependencyIDConnectivity, Name: "Connectivity", Status: health.CheckStatusOK},
				}},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			assert.JSONEq(t, `{
				"capabilities": [
					{
						"name": "Deployment",
						"status": "ok",
						"checks": [
							{
								"name": "Topo",
								"location": "host",
								"status": "ok",
								"value": ""
							},
							{
								"name": "Connectivity",
								"location": "target",
								"status": "ok",
								"value": ""
							}
						]
					},
					{
						"name": "Project management",
						"status": "ok",
						"checks": [
							{
								"name": "Connectivity",
								"location": "target",
								"status": "ok",
								"value": ""
							}
						]
					}
				]
			}`, out.String())
		})

		t.Run("formats blocker references in the value", func(t *testing.T) {
			toPrint := views.HealthReport{Deployment: health.ReadinessReport{Checks: []health.DependencyReport{{
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
				"capabilities": [
					{
						"name": "Deployment",
						"status": "undetermined",
						"checks": [
							{
								"name": "Docker daemon",
								"location": "host",
								"status": "undetermined",
								"value": "not checked: requires host's Docker CLI"
							}
						]
					}
				]
			}`, out.String())
		})
	})
}
