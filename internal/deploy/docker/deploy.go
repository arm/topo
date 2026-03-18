package docker

import (
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

func SupportsRegistry(noRegistry bool, targetHost ssh.Destination) bool {
	return !noRegistry && !targetHost.IsPlainLocalhost()
}

func SupportsSSHControlSockets(goos string) bool {
	return goos != "windows"
}

func NewDeploymentStop(composeFile string, targetHost ssh.Destination) goperation.Sequence {
	return goperation.Sequence{operation.NewDockerComposeStop(composeFile, targetHost)}
}

func NewDeployment(composeFile string, opts DeployOptions) (goperation.Sequence, goperation.Operation) {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}

	var cleanup goperation.Operation
	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.Registry != nil {
			start, securityCheck, stop := ssh.NewSSHTunnel(opts.TargetHost, opts.Registry.Port, opts.Registry.UseControlSockets)
			cleanup = stop
			ops = append(ops, operation.NewRunRegistry(opts.Registry.Port)...)
			ops = append(ops, start)
			ops = append(ops, securityCheck)
			ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, opts.TargetHost, opts.Registry.Port))
			ops = append(ops, stop)
		} else {
			ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, opts.TargetHost))
		}
	}
	ops = append(ops, operation.NewDockerComposeUp(composeFile, opts.TargetHost, opts.RecreateMode))
	return goperation.NewSequence(ops...), cleanup
}
