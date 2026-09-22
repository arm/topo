package health

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

type Engine string

const (
	EngineDocker Engine = "docker"
	EnginePodman Engine = "podman"
)

type HealthCheckOptions struct {
	Engine                  Engine
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
	Topo             Dependency
	SSH              Dependency
	DockerCLI        Dependency
	Docker           Dependency
	DockerCompose    Dependency
	PodmanCLI        Dependency
	PodmanConnection Dependency
	PodmanCompose    Dependency
}

type TargetChecks struct {
	Connectivity          Dependency
	Docker                Dependency
	Remoteproc            Dependency
	RemoteprocRuntime     Dependency
	RemoteprocRuntimeShim Dependency
	PodmanCLI             Dependency
	SSHForwardToPodmanAPI Dependency
	RemotePodmanAPI       Dependency
	Hardware              Dependency
}

type HealthCheck struct {
	Deployment       ReadinessCheck
	ProjectDiscovery ReadinessCheck
}

func NewHealthCheck(options HealthCheckOptions) HealthCheck {
	return AssembleHealthCheck(options.Engine, options.Target, newChecks(options))
}

func AssembleHealthCheck(engine Engine, target *ssh.Destination, checks Checks) HealthCheck {
	registry := NewDependencyRegistry()
	engine = normalizeEngine(engine)
	isLocalTarget := target != nil && target.IsPlainLocalhost()
	hostNodes := registerHostChecks(registry, engine, checks.Host, isLocalTarget)
	targetNodes := registerTargetChecks(registry, engine, target, checks.Target, hostNodes)

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
		Topo:             NewDependencyOnTopo(options.SkipVersionChecks),
		SSH:              NewDependencyOnSSH(localRunner),
		DockerCLI:        NewDependencyOnDockerCLI(localRunner),
		Docker:           NewDependencyOnDockerDaemon(localRunner),
		DockerCompose:    NewDependencyOnDockerCompose(localRunner),
		PodmanCLI:        NewDependencyOnPodmanCLI(localRunner),
		PodmanConnection: NewDependencyOnPodmanConnection(localRunner),
		PodmanCompose:    NewDependencyOnDockerComposeForPodman(probe.CheckPodmanComposeProvider),
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
		PodmanCLI:             NewDependencyOnPodmanCLI(targetRunner),
		SSHForwardToPodmanAPI: NewDependencyOnSSHForwardToPodmanAPI(func(ctx context.Context) probe.RemotePodmanProbeResult {
			return probe.CheckRemotePodmanForwarding(ctx, target)
		}),
		RemotePodmanAPI: NewDependencyOnRemotePodmanAPI(func(ctx context.Context) probe.RemotePodmanProbeResult {
			return probe.CheckRemotePodmanAPI(ctx, target)
		}),
	}
	return checks
}

func normalizeEngine(engine Engine) Engine {
	if engine == EnginePodman {
		return EnginePodman
	}
	return EngineDocker
}

type hostNodes struct {
	deployment []*DependencyNode
	discovery  []*DependencyNode

	dockerCLI     *DependencyNode
	docker        *DependencyNode
	podmanCLI     *DependencyNode
	podmanCompose *DependencyNode
}

func registerHostChecks(registry *DependencyRegistry, engine Engine, checks HostChecks, isLocalTarget bool) hostNodes {
	topo := registry.Register(checks.Topo, DependencyRequirements{}, DependencyScopeHost)
	if engine == EnginePodman {
		return registerPodmanHostChecks(registry, checks, topo, isLocalTarget)
	}
	return registerDockerHostChecks(registry, checks, topo, isLocalTarget)
}

func registerDockerHostChecks(registry *DependencyRegistry, checks HostChecks, topo *DependencyNode, isLocalTarget bool) hostNodes {
	dockerCLI := registry.Register(checks.DockerCLI, DependencyRequirements{}, DependencyScopeHost)
	docker := registry.Register(checks.Docker, DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI}}, DependencyScopeHost)
	compose := registry.Register(checks.DockerCompose, DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI}}, DependencyScopeHost)
	nodes := hostNodes{dockerCLI: dockerCLI, docker: docker}
	if isLocalTarget {
		nodes.deployment = []*DependencyNode{topo, dockerCLI, docker, compose}
	} else {
		ssh := registry.Register(checks.SSH, DependencyRequirements{}, DependencyScopeHost)
		nodes.deployment = []*DependencyNode{topo, ssh, dockerCLI, docker, compose}
		nodes.discovery = []*DependencyNode{ssh}
	}
	return nodes
}

func registerPodmanHostChecks(registry *DependencyRegistry, checks HostChecks, topo *DependencyNode, isLocalTarget bool) hostNodes {
	podmanCLI := registry.Register(checks.PodmanCLI, DependencyRequirements{}, DependencyScopeHost)
	connection := registry.Register(checks.PodmanConnection, DependencyRequirements{Prerequisites: []*DependencyNode{podmanCLI}}, DependencyScopeHost)
	compose := registry.Register(checks.PodmanCompose, DependencyRequirements{Prerequisites: []*DependencyNode{connection, podmanCLI}}, DependencyScopeHost)
	nodes := hostNodes{podmanCLI: podmanCLI, podmanCompose: compose}
	if isLocalTarget {
		nodes.deployment = []*DependencyNode{topo, podmanCLI, connection, compose}
	} else {
		ssh := registry.Register(checks.SSH, DependencyRequirements{}, DependencyScopeHost)
		nodes.deployment = []*DependencyNode{topo, ssh, podmanCLI, connection, compose}
		nodes.discovery = []*DependencyNode{ssh}
	}
	return nodes
}

type targetNodes struct {
	deployment []*DependencyNode
	discovery  []*DependencyNode
}

func registerTargetChecks(registry *DependencyRegistry, engine Engine, target *ssh.Destination, checks TargetChecks, host hostNodes) targetNodes {
	switch {
	case target == nil:
		return targetNodes{}
	case target.IsPlainLocalhost() && engine == EnginePodman:
		return registerLocalTargetPodmanChecks(registry, checks)
	case target.IsPlainLocalhost() && engine == EngineDocker:
		return registerLocalTargetDockerChecks(registry, checks, host.docker)
	case engine == EnginePodman:
		return registerRemotePodmanTargetChecks(registry, checks, host)
	case engine == EngineDocker:
		return registerRemoteDockerTargetChecks(registry, checks, host.dockerCLI)
	default:
		return targetNodes{}
	}
}

func registerRemoteDockerTargetChecks(registry *DependencyRegistry, checks TargetChecks, dockerCLI *DependencyNode) targetNodes {
	access := registry.Register(checks.Connectivity, DependencyRequirements{}, DependencyScopeTarget)
	docker := registry.Register(checks.Docker, DependencyRequirements{Prerequisites: []*DependencyNode{dockerCLI, access}}, DependencyScopeTarget)
	remoteproc := registry.Register(checks.Remoteproc, DependencyRequirements{Prerequisites: []*DependencyNode{access}}, DependencyScopeTarget)
	runtimeRequirements := DependencyRequirements{Conditions: []*DependencyNode{remoteproc}, Prerequisites: []*DependencyNode{docker, access}}
	runtime := registry.Register(checks.RemoteprocRuntime, runtimeRequirements, DependencyScopeTarget)
	shim := registry.Register(checks.RemoteprocRuntimeShim, runtimeRequirements, DependencyScopeTarget)
	hardware := registry.Register(checks.Hardware, DependencyRequirements{Prerequisites: []*DependencyNode{access}}, DependencyScopeTarget)
	return targetNodes{deployment: []*DependencyNode{access, docker, remoteproc, runtime, shim}, discovery: []*DependencyNode{access, hardware}}
}

func registerRemotePodmanTargetChecks(registry *DependencyRegistry, checks TargetChecks, host hostNodes) targetNodes {
	access := registry.Register(checks.Connectivity, DependencyRequirements{}, DependencyScopeTarget)
	podmanCLI := registry.Register(checks.PodmanCLI, DependencyRequirements{Prerequisites: []*DependencyNode{access}}, DependencyScopeTarget)
	tunnel := registry.Register(checks.SSHForwardToPodmanAPI, DependencyRequirements{Prerequisites: []*DependencyNode{podmanCLI}}, DependencyScopeTarget)
	remotePodmanAPI := registry.Register(checks.RemotePodmanAPI, DependencyRequirements{Prerequisites: []*DependencyNode{host.podmanCLI, host.podmanCompose, access, podmanCLI, tunnel}}, DependencyScopeTarget)
	hardware := registry.Register(checks.Hardware, DependencyRequirements{Prerequisites: []*DependencyNode{access}}, DependencyScopeTarget)
	return targetNodes{deployment: []*DependencyNode{access, podmanCLI, tunnel, remotePodmanAPI}, discovery: []*DependencyNode{access, hardware}}
}

func registerLocalTargetDockerChecks(registry *DependencyRegistry, checks TargetChecks, docker *DependencyNode) targetNodes {
	hardware := registry.Register(checks.Hardware, DependencyRequirements{}, DependencyScopeTarget)
	remoteproc := registry.Register(checks.Remoteproc, DependencyRequirements{}, DependencyScopeTarget)
	runtimeRequirements := DependencyRequirements{Conditions: []*DependencyNode{remoteproc}, Prerequisites: []*DependencyNode{docker}}
	runtime := registry.Register(checks.RemoteprocRuntime, runtimeRequirements, DependencyScopeTarget)
	shim := registry.Register(checks.RemoteprocRuntimeShim, runtimeRequirements, DependencyScopeTarget)
	return targetNodes{deployment: []*DependencyNode{remoteproc, runtime, shim}, discovery: []*DependencyNode{hardware}}
}

func registerLocalTargetPodmanChecks(registry *DependencyRegistry, checks TargetChecks) targetNodes {
	hardware := registry.Register(checks.Hardware, DependencyRequirements{}, DependencyScopeTarget)
	return targetNodes{discovery: []*DependencyNode{hardware}}
}
