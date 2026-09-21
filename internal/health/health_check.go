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
	isLocalTarget := target != nil && target.IsPlainLocalhost()
	hostNodes := registerHostChecks(registry, checks.Host, isLocalTarget)
	targetNodes := registerTargetChecks(registry, target, checks.Target, hostNodes)

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
		if evaluation.State == EvaluationOmitted {
			continue
		}
		dependency := reference.Dependency()
		evaluated.Dependencies = append(evaluated.Dependencies, EvaluatedDependency{
			Scope:      reference.Scope(),
			ID:         dependency.ID,
			Label:      dependency.Label,
			Evaluation: evaluation,
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
	Scope      DependencyScope
	ID         DependencyID
	Label      string
	Evaluation DependencyEvaluation
}

func newChecks(options HealthCheckOptions) Checks {
	localRunner := runner.NewLocal()
	checks := Checks{Host: HostChecks{
		Topo:          NewDependencyOnTopo(options.SkipVersionChecks),
		SSH:           NewDependencyOnSSH(localRunner),
		DockerCLI:     NewDependencyOnDockerCLI(localRunner),
		Docker:        NewDependencyOnDockerDaemon(localRunner),
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
		Docker: NewDependencyOnRemoteDockerDaemon(target, func(ctx context.Context, target ssh.Destination) error {
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
	docker     *DependencyNode
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerHostChecks(registry *DependencyRegistry, checks HostChecks, isLocalTarget bool) hostNodes {
	topo := registry.Register(checks.Topo, DependencyRequirements{}, DependencyScopeHost)
	dockerCLI := registry.Register(checks.DockerCLI, DependencyRequirements{}, DependencyScopeHost)
	docker := registry.Register(checks.Docker, DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI}}, DependencyScopeHost)
	compose := registry.Register(
		checks.DockerCompose,
		DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI}},
		DependencyScopeHost,
	)

	nodes := hostNodes{
		dockerCLI:  dockerCLI,
		docker:     docker,
		deployment: []*DependencyNode{topo, dockerCLI, docker, compose},
	}
	if isLocalTarget {
		return nodes
	}

	ssh := registry.Register(checks.SSH, DependencyRequirements{}, DependencyScopeHost)
	nodes.deployment = append(nodes.deployment, ssh)
	nodes.discovery = []*DependencyNode{ssh}
	return nodes
}

type targetNodes struct {
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerTargetChecks(registry *DependencyRegistry, target *ssh.Destination, checks TargetChecks, host hostNodes) targetNodes {
	switch {
	case target == nil:
		return targetNodes{}
	case target.IsPlainLocalhost():
		return registerLocalTargetChecks(registry, checks, host.docker)
	default:
		return registerRemoteTargetChecks(registry, checks, host.dockerCLI)
	}
}

func registerRemoteTargetChecks(registry *DependencyRegistry, checks TargetChecks, dockerCLI *DependencyNode) targetNodes {
	access := registry.Register(checks.Connectivity, DependencyRequirements{}, DependencyScopeTarget)
	hardware := registry.Register(
		checks.Hardware,
		DependencyRequirements{Prerequisites: []*DependencyNode{access}},
		DependencyScopeTarget,
	)
	containerEngineChecks := registerTargetContainerEngineChecks(registry, checks, dockerCLI, access)

	return targetNodes{
		deployment: append([]*DependencyNode{access}, containerEngineChecks...),
		discovery:  []*DependencyNode{access, hardware},
	}
}

func registerLocalTargetChecks(registry *DependencyRegistry, checks TargetChecks, docker *DependencyNode) targetNodes {
	remoteproc := registry.Register(checks.Remoteproc, DependencyRequirements{}, DependencyScopeTarget)
	runtimeRequirements := DependencyRequirements{
		Conditions:    []*DependencyNode{remoteproc},
		Prerequisites: []*DependencyNode{docker},
	}
	runtime := registry.Register(checks.RemoteprocRuntime, runtimeRequirements, DependencyScopeTarget)
	shim := registry.Register(checks.RemoteprocRuntimeShim, runtimeRequirements, DependencyScopeTarget)
	hardware := registry.Register(checks.Hardware, DependencyRequirements{}, DependencyScopeTarget)

	return targetNodes{
		deployment: []*DependencyNode{remoteproc, runtime, shim},
		discovery:  []*DependencyNode{hardware},
	}
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
