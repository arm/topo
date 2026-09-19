package health_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/health"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("evaluates deployment and project discovery checks", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			deployment := health.Dependency{ID: "deployment", Check: passingCheck}
			deploymentRef := registry.Register(deployment)
			projectDiscovery := health.Dependency{ID: "project-discovery", Check: failingCheck}
			projectDiscoveryRef := registry.Register(projectDiscovery)
			healthCheck := health.HealthCheck{
				Deployment:       health.ReadinessCheck{Registry: registry, Host: []*health.DependencyNode{deploymentRef}},
				ProjectDiscovery: health.ReadinessCheck{Registry: registry, Target: []*health.DependencyNode{projectDiscoveryRef}},
			}

			got := healthCheck.Evaluate(context.Background())

			want := health.EvaluatedHealthCheck{
				Deployment: health.EvaluatedReadinessCheck{
					Host: []health.EvaluatedDependency{{
						ID:     deployment.ID,
						Label:  deployment.Label,
						Result: deployment.Check(context.Background()),
					}},
					Target: []health.EvaluatedDependency{},
				},
				ProjectDiscovery: health.EvaluatedReadinessCheck{
					Host: []health.EvaluatedDependency{},
					Target: []health.EvaluatedDependency{{
						ID:     projectDiscovery.ID,
						Label:  projectDiscovery.Label,
						Result: projectDiscovery.Check(context.Background()),
					}},
				},
			}
			assert.Equal(t, want, got)
		})
	})
}

func TestAssembleHealthCheck(t *testing.T) {
	newPassingChecks := func() health.Checks {
		passing := func(label string) health.Dependency {
			return health.Dependency{Label: label, Check: passingCheck}
		}
		return health.Checks{
			Host: health.HostChecks{
				Topo: passing("Topo"), SSH: passing("OpenSSH"), Docker: passing("Container Engine"), DockerCompose: passing("Docker Compose"),
			},
			Target: health.TargetChecks{
				Docker: passing("Target Docker"), Connectivity: passing("Connectivity"), MissingForDeployment: passing("Connectivity"),
				MissingForProjectDiscovery: passing("Connectivity"), Hardware: passing("Hardware Info"), Remoteproc: passing("Remoteproc"),
				RemoteprocRuntime: passing("Remoteproc Runtime"), RemoteprocRuntimeShim: passing("Remoteproc Shim"),
			},
		}
	}

	t.Run("reports distinct missing target dependencies", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.MissingForDeployment = health.Dependency{Label: "Connectivity", Check: failingCheck}
		checks.Target.MissingForProjectDiscovery = health.Dependency{Label: "Connectivity", Check: func(context.Context) health.DependencyCheckResult {
			return health.DependencyCheckResult{Failure: &health.DependencyCheckFailure{
				Severity: health.SeverityWarning,
				Message:  "target unavailable for discovery",
			}}
		}}
		healthCheck := health.AssembleHealthCheck(nil, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentResults := []health.EvaluatedDependency{
			{Label: "Connectivity", Result: checks.Target.MissingForDeployment.Check(context.Background())},
		}
		wantDiscoveryResults := []health.EvaluatedDependency{
			{Label: "Connectivity", Result: checks.Target.MissingForProjectDiscovery.Check(context.Background())},
		}
		assert.Equal(t, wantDeploymentResults, got.Deployment.Target)
		assert.Equal(t, wantDiscoveryResults, got.ProjectDiscovery.Target)
	})

	t.Run("bypasses connectivity for plain localhost", func(t *testing.T) {
		checks := newPassingChecks()
		target := ssh.NewDestination("localhost")
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentResults := []health.EvaluatedDependency{
			{Label: "Target Docker", Result: checks.Target.Docker.Check(context.Background())},
			{Label: "Remoteproc", Result: checks.Target.Remoteproc.Check(context.Background())},
			{Label: "Remoteproc Runtime", Result: checks.Target.RemoteprocRuntime.Check(context.Background())},
			{Label: "Remoteproc Shim", Result: checks.Target.RemoteprocRuntimeShim.Check(context.Background())},
		}
		wantDiscoveryResults := []health.EvaluatedDependency{
			{Label: "Hardware Info", Result: checks.Target.Hardware.Check(context.Background())},
		}
		assert.Equal(t, wantDeploymentResults, got.Deployment.Target)
		assert.Equal(t, wantDiscoveryResults, got.ProjectDiscovery.Target)
	})

	t.Run("shares connectivity and suppresses its dependent target checks", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Connectivity.Check = failingCheck
		target := ssh.NewDestination("user@example.com")
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantResults := []health.EvaluatedDependency{
			{Label: "Connectivity", Result: checks.Target.Connectivity.Check(context.Background())},
		}
		assert.Equal(t, wantResults, got.Deployment.Target)
		assert.Equal(t, wantResults, got.ProjectDiscovery.Target)
	})

	t.Run("suppresses runtime checks when remoteproc fails", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Remoteproc.Check = failingCheck
		target := ssh.NewDestination("localhost")
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentResults := []health.EvaluatedDependency{
			{Label: "Target Docker", Result: checks.Target.Docker.Check(context.Background())},
			{Label: "Remoteproc", Result: checks.Target.Remoteproc.Check(context.Background())},
		}
		assert.Equal(t, wantDeploymentResults, got.Deployment.Target)
	})

	t.Run("keeps hardware discovery independent from container engine readiness", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Docker.Check = failingCheck
		target := ssh.NewDestination("localhost")
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentResults := []health.EvaluatedDependency{
			{Label: "Target Docker", Result: checks.Target.Docker.Check(context.Background())},
			{Label: "Remoteproc", Result: checks.Target.Remoteproc.Check(context.Background())},
		}
		wantDiscoveryResults := []health.EvaluatedDependency{
			{Label: "Hardware Info", Result: checks.Target.Hardware.Check(context.Background())},
		}
		assert.Equal(t, wantDeploymentResults, got.Deployment.Target)
		assert.Equal(t, wantDiscoveryResults, got.ProjectDiscovery.Target)
	})
}

func TestReadinessCheck(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("reports successful dependencies in the group", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			virus := health.Dependency{ID: "virus", Check: passingCheck}
			virusRef := registry.Register(virus)
			bartek := health.Dependency{ID: "bartek", Check: passingCheck}
			bartekRef := registry.Register(bartek, virusRef)

			healthCheck := health.ReadinessCheck{
				Registry: registry,
				Host:     []*health.DependencyNode{bartekRef, virusRef},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{ID: bartek.ID, Label: bartek.Label, Result: bartek.Check(context.Background())},
				{ID: virus.ID, Label: virus.Label, Result: virus.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})

		t.Run("reports a failed prerequisite and omits its dependent", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			flour := health.Dependency{ID: "flour", Check: failingCheck}
			flourRef := registry.Register(flour)
			pizza := health.Dependency{ID: "pizza", Check: passingCheck}
			pizzaRef := registry.Register(pizza, flourRef)

			healthCheck := health.ReadinessCheck{
				Registry: registry,
				Host:     []*health.DependencyNode{flourRef, pizzaRef},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{ID: flour.ID, Label: flour.Label, Result: flour.Check(context.Background())},
			}
			assert.Equal(t, want, got.Host)
		})
	})
}
