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
					Scope:      health.DependencyScopeHost,
					ID:         deployment.ID,
					Label:      deployment.Label,
					Evaluation: health.DependencyEvaluation{Result: deployment.Check(context.Background())},
				}}},
				ProjectDiscovery: health.EvaluatedReadinessCheck{Dependencies: []health.EvaluatedDependency{{
					Scope:      health.DependencyScopeTarget,
					ID:         projectDiscovery.ID,
					Label:      projectDiscovery.Label,
					Evaluation: health.DependencyEvaluation{Result: projectDiscovery.Check(context.Background())},
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

	t.Run("reports dependencies in display order", func(t *testing.T) {
		checks := newPassingChecks()
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentHostDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Host.Topo, State: health.EvaluationExecuted},
			{Dependency: checks.Host.SSH, State: health.EvaluationExecuted},
			{Dependency: checks.Host.DockerCLI, State: health.EvaluationExecuted},
			{Dependency: checks.Host.Docker, State: health.EvaluationExecuted},
			{Dependency: checks.Host.DockerCompose, State: health.EvaluationExecuted},
		}
		wantDeploymentTargetDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Docker, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntime, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntimeShim, State: health.EvaluationExecuted},
		}
		wantDiscoveryHostDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Host.SSH, State: health.EvaluationExecuted},
		}
		wantDiscoveryTargetDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Hardware, State: health.EvaluationExecuted},
		}
		assertEvaluatedDependencies(t, wantDeploymentHostDependencies, hostDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDeploymentTargetDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryHostDependencies, hostDependencies(got.ProjectDiscovery.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryTargetDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
	})

	t.Run("does not report connectivity nor docker engine for a plain localhost target", func(t *testing.T) {
		checks := newPassingChecks()
		localhost := ssh.NewDestination("localhost")
		healthCheck := health.AssembleHealthCheck(&localhost, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Docker, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntime, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntimeShim, State: health.EvaluationExecuted},
		}
		wantDiscoveryDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Hardware, State: health.EvaluationExecuted},
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
	})

	t.Run("reports host capabilities without SSH or target Docker for a plain localhost target", func(t *testing.T) {
		checks := newPassingChecks()
		localhost := ssh.NewDestination("localhost")
		healthCheck := health.AssembleHealthCheck(&localhost, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Host.Topo, State: health.EvaluationExecuted},
			{Dependency: checks.Host.DockerCLI, State: health.EvaluationExecuted},
			{Dependency: checks.Host.Docker, State: health.EvaluationExecuted},
			{Dependency: checks.Host.DockerCompose, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntime, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntimeShim, State: health.EvaluationExecuted},
		}
		wantDiscoveryDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Hardware, State: health.EvaluationExecuted},
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, got.Deployment.Dependencies)
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, got.ProjectDiscovery.Dependencies)
	})

	t.Run("blocks local runtime checks when the host Docker engine fails", func(t *testing.T) {
		checks := newPassingChecks()
		localhost := ssh.NewDestination("localhost")
		checks.Host.Docker.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&localhost, checks)

		got := healthCheck.Evaluate(context.Background())

		dependencies := targetDependencies(got.Deployment.Dependencies)
		want := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntime, State: health.EvaluationBlocked},
			{Dependency: checks.Target.RemoteprocRuntimeShim, State: health.EvaluationBlocked},
		}
		assertEvaluatedDependencies(t, want, dependencies)
	})

	t.Run("shares connectivity and suppresses its dependent target checks", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Connectivity.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Docker, State: health.EvaluationBlocked},
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationBlocked},
		}
		wantDiscoveryDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Hardware, State: health.EvaluationBlocked},
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
	})

	t.Run("suppresses runtime checks when remoteproc fails", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Remoteproc.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		want := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Docker, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationExecuted},
		}
		assertEvaluatedDependencies(t, want, targetDependencies(got.Deployment.Dependencies))
	})

	t.Run("keeps hardware discovery independent from container engine readiness", func(t *testing.T) {
		checks := newPassingChecks()
		checks.Target.Docker.Check = failingCheck
		healthCheck := health.AssembleHealthCheck(&target, checks)

		got := healthCheck.Evaluate(context.Background())

		wantDeploymentDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Docker, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Remoteproc, State: health.EvaluationExecuted},
			{Dependency: checks.Target.RemoteprocRuntime, State: health.EvaluationBlocked},
			{Dependency: checks.Target.RemoteprocRuntimeShim, State: health.EvaluationBlocked},
		}
		wantDiscoveryDependencies := []evaluatedDependencyExpectation{
			{Dependency: checks.Target.Connectivity, State: health.EvaluationExecuted},
			{Dependency: checks.Target.Hardware, State: health.EvaluationExecuted},
		}
		assertEvaluatedDependencies(t, wantDeploymentDependencies, targetDependencies(got.Deployment.Dependencies))
		assertEvaluatedDependencies(t, wantDiscoveryDependencies, targetDependencies(got.ProjectDiscovery.Dependencies))
	})
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
				{Scope: health.DependencyScopeHost, ID: bartek.ID, Label: bartek.Label, Evaluation: health.DependencyEvaluation{Result: bartek.Check(context.Background())}},
				{Scope: health.DependencyScopeHost, ID: virus.ID, Label: virus.Label, Evaluation: health.DependencyEvaluation{Result: virus.Check(context.Background())}},
			}
			assert.Equal(t, want, got.Dependencies)
		})

		t.Run("reports a failed prerequisite and its blocked dependent", func(t *testing.T) {
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

			want := []evaluatedDependencyExpectation{
				{Dependency: flour, State: health.EvaluationExecuted},
				{Dependency: pizza, State: health.EvaluationBlocked},
			}
			assertEvaluatedDependencies(t, want, got.Dependencies)
			assert.Equal(t, []*health.DependencyNode{flourRef}, got.Dependencies[1].Evaluation.BlockedBy)
		})
	})
}

type evaluatedDependencyExpectation struct {
	Dependency health.Dependency
	State      health.EvaluationState
}

func assertEvaluatedDependencies(t *testing.T, want []evaluatedDependencyExpectation, got []health.EvaluatedDependency) {
	t.Helper()
	if !assert.Len(t, got, len(want)) {
		return
	}

	for i, expected := range want {
		assert.Equal(t, expected.Dependency.ID, got[i].ID)
		assert.Equal(t, expected.Dependency.Label, got[i].Label)
		assert.Equal(t, expected.State, got[i].Evaluation.State)
		if expected.State == health.EvaluationExecuted {
			assert.Equal(t, expected.Dependency.Check(context.Background()), got[i].Evaluation.Result)
		}
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

func hostDependencies(dependencies []health.EvaluatedDependency) []health.EvaluatedDependency {
	hosts := make([]health.EvaluatedDependency, 0, len(dependencies))
	for _, dependency := range dependencies {
		if dependency.Scope == health.DependencyScopeHost {
			hosts = append(hosts, dependency)
		}
	}
	return hosts
}
