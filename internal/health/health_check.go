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
	Specified             Dependency
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
	return AssembleHealthCheck(newProductionChecks(options))
}

func AssembleHealthCheck(checks Checks) HealthCheck {
	registry := NewDependencyRegistry()
	hostNodes := registerHostChecks(registry, checks.Host)
	targetNodes := registerTargetChecks(registry, checks.Target)

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
	var targetRunner runner.Runner
	if options.Target != nil {
		targetRunner = runner.For(*options.Target)
	}

	return Checks{
		Host: HostChecks{
			Topo:          NewDependencyOnTopo(options.SkipVersionChecks),
			SSH:           NewDependencyOnSSH(localRunner),
			Docker:        NewDependencyOnDocker(localRunner),
			DockerCompose: NewDependencyOnDockerCompose(localRunner),
		},
		Target: TargetChecks{
			Specified: NewTargetSpecifiedDependency(options.Target, options.MissingTargetFixMessage),
			Connectivity: NewConnectivityDependency(options.Target, ConnectivityOperations{
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
			Docker: NewDependencyOnRemoteDocker(options.Target, localRunner, func(ctx context.Context, target ssh.Destination) error {
				return docker.RunCommand(ctx, io.Discard, docker.NewHostFromDestination(target), "info")
			}),
			Hardware:              NewDependencyOnLscpu(targetRunner),
			Remoteproc:            NewDependencyOnRemoteproc(targetRunner),
			RemoteprocRuntime:     NewDependencyOnRemoteprocRuntime(options.Target, targetRunner),
			RemoteprocRuntimeShim: NewDependencyOnRemoteprocRuntimeShim(options.Target, targetRunner),
		},
	}
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

func registerTargetChecks(registry *DependencyRegistry, checks TargetChecks) targetNodes {
	specified := registry.Register(checks.Specified)
	access := registry.Register(checks.Connectivity, specified)
	hardware := registry.Register(checks.Hardware, access)

	return targetNodes{
		deployment: append([]*DependencyNode{specified, access}, registerTargetContainerEngineChecks(registry, checks, access)...),
		discovery:  []*DependencyNode{specified, access, hardware},
	}
}

func registerTargetContainerEngineChecks(registry *DependencyRegistry, checks TargetChecks, prerequisites ...*DependencyNode) []*DependencyNode {
	docker := registry.Register(checks.Docker, prerequisites...)
	remoteproc := registry.Register(checks.Remoteproc, prerequisites...)
	runtimePrerequisites := append([]*DependencyNode{docker, remoteproc}, prerequisites...)
	runtime := registry.Register(checks.RemoteprocRuntime, runtimePrerequisites...)
	shim := registry.Register(checks.RemoteprocRuntimeShim, runtimePrerequisites...)

	return []*DependencyNode{docker, remoteproc, runtime, shim}
}
