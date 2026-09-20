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
			deploymentRef := registry.Register(deployment, health.DependencyRequirements{}, health.DependencyScopeHost)
			projectDiscovery := health.Dependency{ID: "project-discovery", Check: failingCheck}
			projectDiscoveryRef := registry.Register(
				projectDiscovery,
				health.DependencyRequirements{},
				health.DependencyScopeTarget,
			)
			healthCheck := health.HealthCheck{
				Deployment: health.ReadinessCheck{
					Registry:     registry,
					Dependencies: []*health.DependencyNode{deploymentRef},
				},
				ProjectDiscovery: health.ReadinessCheck{
					Registry:     registry,
					Dependencies: []*health.DependencyNode{projectDiscoveryRef},
				},
			}

			got := healthCheck.Evaluate(context.Background())

			want := health.EvaluatedHealthCheck{
				Deployment: health.EvaluatedReadinessCheck{Dependencies: []health.EvaluatedDependency{{
					Scope:  health.DependencyScopeHost,
					ID:     deployment.ID,
					Label:  deployment.Label,
					Result: deployment.Check(context.Background()),
				}}},
				ProjectDiscovery: health.EvaluatedReadinessCheck{Dependencies: []health.EvaluatedDependency{{
					Scope:  health.DependencyScopeTarget,
					ID:     projectDiscovery.ID,
					Label:  projectDiscovery.Label,
					Result: projectDiscovery.Check(context.Background()),
				}}},
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
				Topo: passing("Topo"), SSH: passing("OpenSSH"), DockerCLI: passing("Docker CLI"),
				Docker: passing("Container Engine"), DockerCompose: passing("Docker Compose"),
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

		assert.Empty(t, targetDependencies(got.Deployment.Dependencies))
		assert.Empty(t, targetDependencies(got.ProjectDiscovery.Dependencies))
	})

	t.Run("does not probe the target engine when Docker CLI is unavailable", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Host.DockerCLI.Check = failingCheck
		targetDockerChecks := 0
		checks.Target.Docker.Check = func(context.Context) health.DependencyCheckResult {
			targetDockerChecks++
			return passingCheck(context.Background())
		}
		healthCheck := health.AssembleHealthCheck(&target, checks)

		healthCheck.Evaluate(context.Background())

		assert.Zero(t, targetDockerChecks)
	})

	t.Run("runs target engine and Compose checks when the host daemon fails", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Host.Docker.Check = failingCheck
		composeChecks := 0
		checks.Host.DockerCompose.Check = func(context.Context) health.DependencyCheckResult {
			composeChecks++
			return passingCheck(context.Background())
		}
		targetDockerChecks := 0
		checks.Target.Docker.Check = func(context.Context) health.DependencyCheckResult {
			targetDockerChecks++
			return passingCheck(context.Background())
		}
		healthCheck := health.AssembleHealthCheck(&target, checks)

		healthCheck.Evaluate(context.Background())

		assert.Equal(t, 1, composeChecks)
		assert.Equal(t, 1, targetDockerChecks)
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
		assertEvaluatedDependencies(t, wantDeploymentDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
	})

	t.Run("does not report connectivity for a plain localhost target", func(t *testing.T) {
		checks := newPassingChecks()
		localhost := ssh.NewDestination("localhost")
		healthCheck := health.AssembleHealthCheck(&localhost, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []health.Dependency{
			checks.Target.Docker,
			checks.Target.Remoteproc,
			checks.Target.RemoteprocRuntime,
			checks.Target.RemoteprocRuntimeShim,
		}
		wantDiscoveryDependencies := []health.Dependency{
			checks.Target.Hardware,
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
	})

	t.Run("shares connectivity and suppresses its dependent target checks", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Connectivity.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		want := []health.Dependency{
			checks.Target.Connectivity,
		}
		assertEvaluatedDependencies(t, want, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, want, targetDependencies(got.ProjectDiscovery.Dependencies))
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
		assertEvaluatedDependencies(t, want, targetDependencies(got.Deployment.Dependencies))
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
		assertEvaluatedDependencies(t, wantDeploymentDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
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
			Scope:  health.DependencyScopeTarget,
			ID:     dependency.ID,
			Label:  dependency.Label,
			Result: wantResult,
		}
		assert.Equal(t, wantDependency, got[i])
	}
}

func targetDependencies(dependencies []health.EvaluatedDependency) []health.EvaluatedDependency {
	targets := make([]health.EvaluatedDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		if dependency.Scope == health.DependencyScopeTarget {
			targets = append(targets, dependency)
		}
	}
	return targets
}

func TestReadinessCheck(t *testing.T) {
	t.Run("Evaluate", func(t *testing.T) {
		t.Run("reports successful dependencies in the group", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			virus := health.Dependency{ID: "virus", Check: passingCheck}
			virusRef := registry.Register(virus, health.DependencyRequirements{}, health.DependencyScopeHost)
			bartek := health.Dependency{ID: "bartek", Check: passingCheck}
			bartekRef := registry.Register(
				bartek,
				health.DependencyRequirements{Prerequisites: []*health.DependencyNode{virusRef}},
				health.DependencyScopeHost,
			)

			healthCheck := health.ReadinessCheck{
				Registry:     registry,
				Dependencies: []*health.DependencyNode{bartekRef, virusRef},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{Scope: health.DependencyScopeHost, ID: bartek.ID, Label: bartek.Label, Result: bartek.Check(context.Background())},
				{Scope: health.DependencyScopeHost, ID: virus.ID, Label: virus.Label, Result: virus.Check(context.Background())},
			}
			assert.Equal(t, want, got.Dependencies)
		})

		t.Run("reports a failed prerequisite and omits its dependent", func(t *testing.T) {
			registry := health.NewDependencyRegistry()
			flour := health.Dependency{ID: "flour", Check: failingCheck}
			flourRef := registry.Register(flour, health.DependencyRequirements{}, health.DependencyScopeHost)
			pizza := health.Dependency{ID: "pizza", Check: passingCheck}
			pizzaRef := registry.Register(
				pizza,
				health.DependencyRequirements{Prerequisites: []*health.DependencyNode{flourRef}},
				health.DependencyScopeHost,
			)

			healthCheck := health.ReadinessCheck{
				Registry:     registry,
				Dependencies: []*health.DependencyNode{flourRef, pizzaRef},
			}

			got := healthCheck.Evaluate(context.Background())

			want := []health.EvaluatedDependency{
				{Scope: health.DependencyScopeHost, ID: flour.ID, Label: flour.Label, Result: flour.Check(context.Background())},
			}
			assert.Equal(t, want, got.Dependencies)
		})
	})
}
