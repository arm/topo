package docker

import (
	"github.com/arm/topo/internal/deploy/docker/operation"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
)

type DeployOptions struct {
	RecreateMode         operation.RecreateMode
	WithRegistry         bool
	TargetHost           ssh.SSHConnection
	RegistryPort         string
	UseSSHControlSockets bool
}

func SupportsRegistry(noRegistry bool, targetHost ssh.SSHConnection) bool {
	return !noRegistry && !targetHost.IsLocalhost()
}

func SupportsSSHControlSockets(goos string) bool {
	return goos != "windows"
}

func NewDeploymentStop(composeFile string, targetHost ssh.SSHConnection) goperation.Sequence {
	return goperation.Sequence{operation.NewDockerComposeStop(composeFile, targetHost)}
}

func NewDeployment(composeFile string, opts DeployOptions) (goperation.Sequence, goperation.Operation) {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}

	var cleanup goperation.Operation
	if !opts.TargetHost.IsLocalhost() {
		if opts.WithRegistry {
			start, securityCheck, stop := ssh.NewSSHTunnel(opts.TargetHost, opts.RegistryPort, opts.UseSSHControlSockets)
			if start == nil {
				return nil, nil
			}
			cleanup = stop
			ops = append(ops, operation.NewRunRegistry(opts.RegistryPort)...)
			ops = append(ops, start)
			ops = append(ops, securityCheck)
			ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, opts.TargetHost, opts.RegistryPort))
			ops = append(ops, stop)
		} else {
			ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, opts.TargetHost))
		}
	}
	ops = append(ops, operation.NewDockerComposeUp(composeFile, opts.TargetHost, opts.RecreateMode))
	return goperation.NewSequence(ops...), cleanup
}
