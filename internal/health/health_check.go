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
	Docker                     Dependency
	Connectivity               Dependency
	MissingForDeployment       Dependency
	MissingForProjectDiscovery Dependency
	Hardware                   Dependency
	Remoteproc                 Dependency
	RemoteprocRuntime          Dependency
	RemoteprocRuntimeShim      Dependency
}

type HealthCheck struct {
	Deployment       ReadinessCheck
	ProjectDiscovery ReadinessCheck
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	return AssembleHealthCheck(options.Target, newProductionChecks(options))
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

func newProductionChecks(options HealthCheckOptions) Checks {
	localRunner := runner.NewLocal()
	checks := Checks{
		Host: HostChecks{
			Topo:          NewDependencyOnTopo(options.SkipVersionChecks),
			SSH:           NewDependencyOnSSH(localRunner),
			Docker:        NewDependencyOnDocker(localRunner),
			DockerCompose: NewDependencyOnDockerCompose(localRunner),
		},
		Target: TargetChecks{
			MissingForDeployment: NewMissingTargetDependency(
				"target not specified", SeverityError, options.MissingTargetFixMessage,
			),
			MissingForProjectDiscovery: NewMissingTargetDependency(
				"target not specified; cannot calculate project compatibility", SeverityWarning, options.MissingTargetFixMessage,
			),
		},
	}
	if options.Target == nil {
		return checks
	}

	target := *options.Target
	targetRunner := runner.For(target)
	host := docker.NewHostFromDestination(target)
	checks.Target.Docker = NewDependencyOnRemoteDocker(localRunner, func(ctx context.Context) error {
		return docker.RunCommand(ctx, io.Discard, host, "info")
	})
	sshRunner := runner.NewSSH(target)
	checks.Target.Connectivity = NewConnectivityDependency(target, ConnectivityOperations{
		Authenticate: func(ctx context.Context) error {
			return probe.SSHAuthentication(ctx, sshRunner, options.AcceptHostKeys)
		},
		KnownHostsEntry: func() (string, error) {
			config, err := ssh.LoadConfig(target)
			if err != nil {
				return "", err
			}
			return config.AsKnownHostsEntry(), nil
		},
	})
	checks.Target.Hardware = NewDependencyOnLscpu(targetRunner)
	checks.Target.Remoteproc = NewDependencyOnRemoteproc(targetRunner)
	checks.Target.RemoteprocRuntime = NewDependencyOnRemoteprocRuntime(target, targetRunner)
	checks.Target.RemoteprocRuntimeShim = NewDependencyOnRemoteprocRuntimeShim(target, targetRunner)
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
		return targetNodes{
			deployment: []*DependencyNode{registry.Register(checks.MissingForDeployment)},
			discovery:  []*DependencyNode{registry.Register(checks.MissingForProjectDiscovery)},
		}
	}

	prerequisites := []*DependencyNode(nil)
	nodes := targetNodes{}
	if !target.IsPlainLocalhost() {
		connectivity := registry.Register(checks.Connectivity)
		prerequisites = []*DependencyNode{connectivity}
		nodes.deployment = append(nodes.deployment, connectivity)
		nodes.discovery = append(nodes.discovery, connectivity)
	}

	hardware := registry.Register(checks.Hardware, prerequisites...)
	nodes.discovery = append(nodes.discovery, hardware)
	nodes.deployment = append(nodes.deployment, registerTargetContainerEngineChecks(registry, checks, prerequisites...)...)
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
