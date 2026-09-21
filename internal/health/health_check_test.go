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
				Connectivity: passing("Target access"), Docker: passing("Target Docker"),
				Hardware: passing("Hardware Info"), Remoteproc: passing("Remoteproc"), RemoteprocRuntime: passing("Remoteproc Runtime"),
				RemoteprocRuntimeShim: passing("Remoteproc Shim"),
			},
		}
	}
	target := ssh.NewDestination("user@example.com")

	t.Run("does not register target checks when target is not specified", func(t *testing.T) {
		checks := newPassingChecks()
		healthCheck := health.AssembleHealthCheck(nil, checks)

		got := healthCheck.Evaluate(context.Background())

		assert.Empty(t, got.Deployment.Target)
		assert.Empty(t, got.ProjectDiscovery.Target)
	})

	t.Run("runs target checks after their prerequisites succeed", func(t *testing.T) {
		checks := newPassingChecks()
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []health.Dependency{
			checks.Target.Connectivity,
			checks.Target.Docker,
			checks.Target.Remoteproc,
			checks.Target.RemoteprocRuntime,
			checks.Target.RemoteprocRuntimeShim,
		}
		wantDiscoveryDependencies := []health.Dependency{
			checks.Target.Connectivity,
			checks.Target.Hardware,
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, got.Deployment.Target)
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, got.ProjectDiscovery.Target)
	})

	t.Run("shares connectivity and suppresses its dependent target checks", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Connectivity.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		want := []health.Dependency{
			checks.Target.Connectivity,
		}
		assertEvaluatedDependencies(t, want, got.Deployment.Target)
		assertEvaluatedDependencies(t, want, got.ProjectDiscovery.Target)
	})

	t.Run("suppresses runtime checks when remoteproc fails", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Remoteproc.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		want := []health.Dependency{
			checks.Target.Connectivity,
			checks.Target.Docker,
			checks.Target.Remoteproc,
		}
		assertEvaluatedDependencies(t, want, got.Deployment.Target)
	})

	t.Run("keeps hardware discovery independent from container engine readiness", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Docker.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []health.Dependency{
			checks.Target.Connectivity,
			checks.Target.Docker,
			checks.Target.Remoteproc,
		}
		wantDiscoveryDependencies := []health.Dependency{
			checks.Target.Connectivity,
			checks.Target.Hardware,
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, got.Deployment.Target)
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, got.ProjectDiscovery.Target)
	})
}

func assertEvaluatedDependencies(t *testing.T, want []health.Dependency, got []health.EvaluatedDependency) {
	t.Helper()
	if !assert.Len(t, got, len(want)) {
		return
	}

	for i, dependency := range want {
		wantResult := dependency.Check(context.Background())
		wantDependency := health.EvaluatedDependency{
			ID:     dependency.ID,
			Label:  dependency.Label,
			Result: wantResult,
		}
		assert.Equal(t, wantDependency, got[i])
	}
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
