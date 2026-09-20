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
	DockerCLI     Dependency
	Docker        Dependency
	DockerCompose Dependency
}

type TargetChecks struct {
	Connectivity          Dependency
	Docker                Dependency
	Remoteproc            Dependency
	RemoteprocRuntime     Dependency
	RemoteprocRuntimeShim Dependency
	Hardware              Dependency
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
	targetNodes := registerTargetChecks(registry, target, checks.Target, hostNodes.dockerCLI)

	return HealthCheck{
		Deployment: ReadinessCheck{
			Registry:     registry,
			Dependencies: append(hostNodes.deployment, targetNodes.deployment...),
		},
		ProjectDiscovery: ReadinessCheck{
			Registry:     registry,
			Dependencies: append(hostNodes.discovery, targetNodes.discovery...),
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
	Registry     *DependencyRegistry
	Dependencies []*DependencyNode
}

func (h ReadinessCheck) Evaluate(ctx context.Context) EvaluatedReadinessCheck {
	evaluated := EvaluatedReadinessCheck{Dependencies: make([]EvaluatedDependency, 0, len(h.Dependencies))}
	for _, reference := range h.Dependencies {
		evaluation := h.Registry.Check(ctx, reference)
		if evaluation.State != EvaluationExecuted {
			continue
		}
		dependency := reference.Dependency()
		evaluated.Dependencies = append(evaluated.Dependencies, EvaluatedDependency{
			Scope:  reference.Scope(),
			ID:     dependency.ID,
			Label:  dependency.Label,
			Result: evaluation.Result,
		})
	}
	return evaluated
}

type EvaluatedHealthCheck struct {
	Deployment       EvaluatedReadinessCheck
	ProjectDiscovery EvaluatedReadinessCheck
}

type EvaluatedReadinessCheck struct {
	Dependencies []EvaluatedDependency
}

type EvaluatedDependency struct {
	Scope  DependencyScope
	ID     DependencyID
	Label  string
	Result DependencyCheckResult
}

func newChecks(options HealthCheckOptions) Checks {
	localRunner := runner.NewLocal()
	checks := Checks{Host: HostChecks{
		Topo:          NewDependencyOnTopo(options.SkipVersionChecks),
		SSH:           NewDependencyOnSSH(localRunner),
		DockerCLI:     NewDependencyOnDockerCLI(localRunner),
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
		Docker: NewDependencyOnRemoteDocker(target, func(ctx context.Context, target ssh.Destination) error {
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
	dockerCLI  *DependencyNode
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerHostChecks(registry *DependencyRegistry, checks HostChecks) hostNodes {
	topo := registry.Register(checks.Topo, DependencyRequirements{}, DependencyScopeHost)
	ssh := registry.Register(checks.SSH, DependencyRequirements{}, DependencyScopeHost)
	dockerCLI := registry.Register(checks.DockerCLI, DependencyRequirements{}, DependencyScopeHost)
	docker := registry.Register(checks.Docker, DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI}}, DependencyScopeHost)
	compose := registry.Register(
		checks.DockerCompose,
		DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI}},
		DependencyScopeHost,
	)

	return hostNodes{
		dockerCLI:  dockerCLI,
		deployment: []*DependencyNode{topo, ssh, dockerCLI, docker, compose},
		discovery:  []*DependencyNode{ssh},
	}
}

type targetNodes struct {
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerTargetChecks(registry *DependencyRegistry, target *ssh.Destination, checks TargetChecks, dockerCLI *DependencyNode) targetNodes {
	if target == nil {
		return targetNodes{}
	}

	prerequisites := []*DependencyNode(nil)
	nodes := targetNodes{}
	if !target.IsPlainLocalhost() {
		access := registry.Register(checks.Connectivity, DependencyRequirements{}, DependencyScopeTarget)
		prerequisites = []*DependencyNode{access}
		nodes.deployment = append(nodes.deployment, access)
		nodes.discovery = append(nodes.discovery, access)
	}

	hardware := registry.Register(
		checks.Hardware,
		DependencyRequirements{Prerequisites: prerequisites},
		DependencyScopeTarget,
	)
	nodes.deployment = append(nodes.deployment, registerTargetContainerEngineChecks(registry, checks, dockerCLI, prerequisites...)...)
	nodes.discovery = append(nodes.discovery, hardware)
	return nodes
}

func registerTargetContainerEngineChecks(registry *DependencyRegistry, checks TargetChecks, dockerCLI *DependencyNode, prerequisites ...*DependencyNode) []*DependencyNode {
	dockerPrerequisites := append([]*DependencyNode{dockerCLI}, prerequisites...)
	docker := registry.Register(
		checks.Docker,
		DependencyRequirements{Prerequisites: dockerPrerequisites},
		DependencyScopeTarget,
	)
	remoteproc := registry.Register(
		checks.Remoteproc,
		DependencyRequirements{Prerequisites: prerequisites},
		DependencyScopeTarget,
	)
	runtimeRequirements := DependencyRequirements{
		Conditions:    []*DependencyNode{remoteproc},
		Prerequisites: append([]*DependencyNode{docker}, prerequisites...),
	}
	runtime := registry.Register(
		checks.RemoteprocRuntime,
		runtimeRequirements,
		DependencyScopeTarget,
	)
	shim := registry.Register(
		checks.RemoteprocRuntimeShim, runtimeRequirements, DependencyScopeTarget)

	return []*DependencyNode{docker, remoteproc, runtime, shim}
}
