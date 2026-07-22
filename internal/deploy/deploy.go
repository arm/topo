package deploy

import (
	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/deploy/post_deploy"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
)

const (
	DefaultRegistryContainerName = "topo-registry"
	DefaultRegistryPort          = "12737"
)

type RegistryConfig struct {
	Port                string
	SkipRemotePortCheck bool
	UseControlSockets   bool
}

type DeployOptions struct {
	RecreateMode docker.RecreateMode
	TargetHost   ssh.Destination
	Registry     *RegistryConfig
}

func SupportsRegistry(noRegistry bool, dest ssh.Destination) bool {
	return !noRegistry && !dest.IsPlainLocalhost()
}

func SupportsSSHControlSockets(goos string) bool {
	return goos != "windows"
}

func NewDeploymentStop(composeFile string, dest ssh.Destination) goperation.Sequence {
	return goperation.Sequence{operation.NewDockerComposeStop(composeFile, command.NewHostFromDestination(dest))}
}

func NewDeployment(composeFile string, opts DeployOptions) (goperation.Sequence, goperation.Operation) {
	sourceHost := command.LocalHost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}

	targetHost := command.NewHostFromDestination(opts.TargetHost)
	var cleanup goperation.Operation
	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry != nil {
			startTunnel, stopTunnel := operation.NewRegistrySSHTunnel(opts.TargetHost, opts.Registry.Port, opts.Registry.UseControlSockets)
			cleanup = stopTunnel
			ops = append(ops, operation.NewRunRegistry(DefaultRegistryContainerName, opts.Registry.Port)...)
			ops = append(ops, startTunnel)
			if !opts.TargetHost.IsLocalhost() && !opts.Registry.SkipRemotePortCheck {
				ops = append(ops, operation.NewRegistryTunnelExposureCheck(opts.TargetHost, opts.Registry.Port))
			}
			ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, targetHost, opts.Registry.Port))
			ops = append(ops, stopTunnel)
		} else {
			ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, targetHost))
		}
	}
	ops = append(ops, operation.NewDockerComposeUp(composeFile, targetHost, opts.RecreateMode))
	ops = append(ops, post_deploy.NewDeploySuccess(composeFile, targetHost, post_deploy.DefaultMessage(composeFile)))
	return goperation.NewSequence(ops...), cleanup
}
