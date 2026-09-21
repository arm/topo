package health

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type HealthCheckOptions struct {
	Target                  *ssh.Destination
	MissingTargetFixMessage string
	SkipVersionChecks       bool
	AcceptHostKeys          bool
}

type Checks struct {
	Host   HostChecks
	Target TargetChecks
}

type HostChecks struct {
	Topo          Dependency
	SSH           Dependency
	Docker        Dependency
	DockerCompose Dependency
}

type TargetChecks struct {
	Connectivity          Dependency
	Docker                Dependency
	Hardware              Dependency
	Remoteproc            Dependency
	RemoteprocRuntime     Dependency
	RemoteprocRuntimeShim Dependency
}

type HealthCheck struct {
	Deployment       ReadinessCheck
	ProjectDiscovery ReadinessCheck
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	return AssembleHealthCheck(options.Target, newChecks(options))
}

func AssembleHealthCheck(target *ssh.Destination, checks Checks) HealthCheck {
	registry := NewDependencyRegistry()
	hostNodes := registerHostChecks(registry, checks.Host)
	targetNodes := registerTargetChecks(registry, target, checks.Target)

	return HealthCheck{
		Deployment: ReadinessCheck{
			Registry: registry,
			Host:     hostNodes.deployment,
			Target:   targetNodes.deployment,
		},
		ProjectDiscovery: ReadinessCheck{
			Registry: registry,
			Host:     hostNodes.discovery,
			Target:   targetNodes.discovery,
		},
	}
}

func (h HealthCheck) Evaluate(ctx context.Context) EvaluatedHealthCheck {
	return EvaluatedHealthCheck{
		Deployment:       h.Deployment.Evaluate(ctx),
		ProjectDiscovery: h.ProjectDiscovery.Evaluate(ctx),
	}
}

type ReadinessCheck struct {
	Registry *DependencyRegistry
	Host     []*DependencyNode
	Target   []*DependencyNode
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
		statuses = append(statuses, EvaluatedDependency{ID: dependency.ID, Label: dependency.Label, Result: result})
	}
	return statuses
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

func newChecks(options HealthCheckOptions) Checks {
	localRunner := runner.NewLocal()
	checks := Checks{Host: HostChecks{
		Topo:          NewDependencyOnTopo(options.SkipVersionChecks),
		SSH:           NewDependencyOnSSH(localRunner),
		Docker:        NewDependencyOnDocker(localRunner),
		DockerCompose: NewDependencyOnDockerCompose(localRunner),
	}}
	if options.Target == nil {
		return checks
	}

	target := *options.Target
	targetRunner := runner.For(target)
	checks.Target = TargetChecks{
		Connectivity: NewConnectivityDependency(target, ConnectivityOperations{
			Authenticate: func(ctx context.Context, target ssh.Destination) error {
				return probe.SSHAuthentication(ctx, runner.NewSSH(target), options.AcceptHostKeys)
			},
			KnownHostsEntry: func(target ssh.Destination) (string, error) {
				config, err := ssh.LoadConfig(target)
				if err != nil {
					return "", err
				}
				return config.AsKnownHostsEntry(), nil
			},
		}),
		Docker: NewDependencyOnRemoteDocker(target, localRunner, func(ctx context.Context, target ssh.Destination) error {
			return docker.RunCommand(ctx, io.Discard, docker.NewHostFromDestination(target), "info")
		}),
		Hardware:              NewDependencyOnLscpu(targetRunner),
		Remoteproc:            NewDependencyOnRemoteproc(targetRunner),
		RemoteprocRuntime:     NewDependencyOnRemoteprocRuntime(target, targetRunner),
		RemoteprocRuntimeShim: NewDependencyOnRemoteprocRuntimeShim(target, targetRunner),
	}
	return checks
}

type hostNodes struct {
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerHostChecks(registry *DependencyRegistry, checks HostChecks) hostNodes {
	topo := registry.Register(checks.Topo)
	ssh := registry.Register(checks.SSH)
	docker := registry.Register(checks.Docker)
	compose := registry.Register(checks.DockerCompose, docker)

	return hostNodes{
		deployment: []*DependencyNode{topo, ssh, docker, compose},
		discovery:  []*DependencyNode{ssh},
	}
}

type targetNodes struct {
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerTargetChecks(registry *DependencyRegistry, target *ssh.Destination, checks TargetChecks) targetNodes {
	if target == nil {
		return targetNodes{}
	}

	prerequisites := []*DependencyNode(nil)
	nodes := targetNodes{}
	if !target.IsPlainLocalhost() {
		access := registry.Register(checks.Connectivity)
		prerequisites = []*DependencyNode{access}
		nodes.deployment = append(nodes.deployment, access)
		nodes.discovery = append(nodes.discovery, access)
	}

	hardware := registry.Register(checks.Hardware, prerequisites...)
	nodes.deployment = append(nodes.deployment, registerTargetContainerEngineChecks(registry, checks, prerequisites...)...)
	nodes.discovery = append(nodes.discovery, hardware)
	return nodes
}

func registerTargetContainerEngineChecks(registry *DependencyRegistry, checks TargetChecks, prerequisites ...*DependencyNode) []*DependencyNode {
	docker := registry.Register(checks.Docker, prerequisites...)
	remoteproc := registry.Register(checks.Remoteproc, prerequisites...)
	runtimePrerequisites := append([]*DependencyNode{docker, remoteproc}, prerequisites...)
	runtime := registry.Register(checks.RemoteprocRuntime, runtimePrerequisites...)
	shim := registry.Register(checks.RemoteprocRuntimeShim, runtimePrerequisites...)

	return []*DependencyNode{docker, remoteproc, runtime, shim}
}
