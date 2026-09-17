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
	Registry *DependencyRegistry
	Host     []*DependencyNode
	Target   []*DependencyNode
}

type EvaluatedDependency struct {
	ID     DependencyID
	Label  string
	Result DependencyCheckResult
}

type EvaluatedHealthCheck struct {
	Host   []EvaluatedDependency
	Target []EvaluatedDependency
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	registry := NewDependencyRegistry()

	dependencyTopo := registry.Register(NewDependencyOnTopo(options.SkipVersionChecks))
	localRunner := runner.NewLocal()
	dependencySSH := registry.Register(NewDependencyOnSSH(localRunner))
	dependencyDocker := registry.Register(NewDependencyOnDocker(localRunner))
	dependencyDockerCompose := registry.Register(NewDependencyOnDockerCompose(localRunner), dependencyDocker)
	hostDependencies := []*DependencyNode{dependencyTopo, dependencySSH, dependencyDocker, dependencyDockerCompose}

	targetPrerequisites := []*DependencyNode(nil)
	targetDependencies := []*DependencyNode(nil)
	if options.Target == nil || !options.Target.IsPlainLocalhost() {
		dependencyConnectivity := registry.Register(NewConnectivityDependency(options.Target, options.AcceptHostKeys, options.MissingTargetFixMessage))
		targetPrerequisites = []*DependencyNode{dependencyConnectivity}
		targetDependencies = append(targetDependencies, dependencyConnectivity)
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
		targetDependencies = append(targetDependencies, dependencyDocker, dependencyRemoteproc, dependencyRemoteprocRuntime, dependencyRemoteprocRuntimeShim, dependencyLscpu)
	}

	return HealthCheck{
		Registry: registry,
		Host:     hostDependencies,
		Target:   targetDependencies,
	}
}

func (h HealthCheck) Evaluate(ctx context.Context) EvaluatedHealthCheck {
	return EvaluatedHealthCheck{
		Host:   h.evaluateDependencies(ctx, h.Host),
		Target: h.evaluateDependencies(ctx, h.Target),
	}
}

func (h HealthCheck) evaluateDependencies(ctx context.Context, references []*DependencyNode) []EvaluatedDependency {
	statuses := make([]EvaluatedDependency, 0, len(references))
	for _, reference := range references {
		result, hasUnmetPrerequisites := h.Registry.Check(ctx, reference)
		if hasUnmetPrerequisites {
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
