package docker

import (
	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/deploy/docker/operation"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
)

type RegistryConfig struct {
	Port              string
	UseControlSockets bool
}

type DeployOptions struct {
	RecreateMode operation.RecreateMode
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
			start, securityCheck, stop := ssh.NewSSHTunnel(opts.TargetHost, opts.Registry.Port, opts.Registry.UseControlSockets)
			cleanup = stop
			ops = append(ops, operation.NewRunRegistry(opts.Registry.Port)...)
			ops = append(ops, start)
			ops = append(ops, securityCheck)
			ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, targetHost, opts.Registry.Port))
			ops = append(ops, stop)
		} else {
			ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, targetHost))
		}
	}
	ops = append(ops, operation.NewDockerComposeUp(composeFile, targetHost, opts.RecreateMode))
	return goperation.NewSequence(ops...), cleanup
}
