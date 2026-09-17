package health

import (
	"context"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type HealthCheckOptions struct {
	Target                  *ssh.Destination
	MissingTargetFixMessage string
	SkipVersionChecks       bool
	AcceptHostKeys          bool
}

type HealthCheck struct {
	Deployment       ReadinessCheck
	ProjectDiscovery ReadinessCheck
}

type ReadinessCheck struct {
	Registry *DependencyRegistry
	Host     []*DependencyNode
	Target   []*DependencyNode
}

type EvaluatedHealthCheck struct {
	Deployment       EvaluatedReadinessCheck
	ProjectDiscovery EvaluatedReadinessCheck
}

type EvaluatedReadinessCheck struct {
	Host   []EvaluatedDependency
	Target []EvaluatedDependency
}

type EvaluatedDependency struct {
	ID     DependencyID
	Label  string
	Result DependencyCheckResult
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	registry := NewDependencyRegistry()

	dependencyTopo := registry.Register(NewDependencyOnTopo(options.SkipVersionChecks))
	localRunner := runner.NewLocal()
	dependencySSH := registry.Register(NewDependencyOnSSH(localRunner))
	dependencyDocker := registry.Register(NewDependencyOnDocker(localRunner))
	dependencyDockerCompose := registry.Register(NewDependencyOnDockerCompose(localRunner), dependencyDocker)
	deploymentHostDependencies := []*DependencyNode{dependencyTopo, dependencySSH, dependencyDocker, dependencyDockerCompose}
	projectDiscoveryHostDependencies := []*DependencyNode{dependencySSH}

	targetPrerequisites := []*DependencyNode(nil)
	deploymentTargetDependencies := []*DependencyNode(nil)
	projectDiscoveryTargetDependencies := []*DependencyNode(nil)
	if options.Target == nil || !options.Target.IsPlainLocalhost() {
		dependencyConnectivity := registry.Register(NewConnectivityDependency(options.Target, options.AcceptHostKeys, options.MissingTargetFixMessage))
		targetPrerequisites = []*DependencyNode{dependencyConnectivity}
		deploymentTargetDependencies = append(deploymentTargetDependencies, dependencyConnectivity)
		projectDiscoveryTargetDependencies = append(projectDiscoveryTargetDependencies, dependencyConnectivity)
	}
	if options.Target != nil {
		targetRunner := runner.For(*options.Target)
		dependencyDocker := registry.Register(NewDependencyOnDocker(targetRunner), targetPrerequisites...)
		dependencyRemoteproc := registry.Register(NewDependencyOnRemoteproc(targetRunner), targetPrerequisites...)
		dependencyRemoteprocRuntime := registry.Register(
			NewDependencyOnRemoteprocRuntime(*options.Target, targetRunner),
			append([]*DependencyNode{dependencyDocker, dependencyRemoteproc}, targetPrerequisites...)...,
		)
		dependencyRemoteprocRuntimeShim := registry.Register(
			NewDependencyOnRemoteprocRuntimeShim(*options.Target, targetRunner),
			append([]*DependencyNode{dependencyDocker, dependencyRemoteproc}, targetPrerequisites...)...,
		)
		dependencyLscpu := registry.Register(NewDependencyOnLscpu(targetRunner), targetPrerequisites...)
		deploymentTargetDependencies = append(deploymentTargetDependencies, dependencyDocker, dependencyRemoteproc, dependencyRemoteprocRuntime, dependencyRemoteprocRuntimeShim)
		projectDiscoveryTargetDependencies = append(projectDiscoveryTargetDependencies, dependencyLscpu)
	}

	return HealthCheck{
		Deployment: ReadinessCheck{
			Registry: registry,
			Host:     deploymentHostDependencies,
			Target:   deploymentTargetDependencies,
		},
		ProjectDiscovery: ReadinessCheck{
			Registry: registry,
			Host:     projectDiscoveryHostDependencies,
			Target:   projectDiscoveryTargetDependencies,
		},
	}
}

func (h HealthCheck) Evaluate(ctx context.Context) EvaluatedHealthCheck {
	return EvaluatedHealthCheck{
		Deployment:       h.Deployment.Evaluate(ctx),
		ProjectDiscovery: h.ProjectDiscovery.Evaluate(ctx),
	}
}

func (h ReadinessCheck) Evaluate(ctx context.Context) EvaluatedReadinessCheck {
	return EvaluatedReadinessCheck{
		Host:   h.evaluateDependencies(ctx, h.Host),
		Target: h.evaluateDependencies(ctx, h.Target),
	}
}

func (h ReadinessCheck) evaluateDependencies(ctx context.Context, references []*DependencyNode) []EvaluatedDependency {
	statuses := make([]EvaluatedDependency, 0, len(references))
	for _, reference := range references {
		result, checked := h.Registry.Check(ctx, reference)
		if !checked {
			continue
		}
		dependency := reference.Dependency()
		statuses = append(statuses, EvaluatedDependency{
			ID:     dependency.ID,
			Label:  dependency.Label,
			Result: result,
		})
	}
	return statuses
}
