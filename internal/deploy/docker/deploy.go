package docker

import (
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/ssh"
)

type DeployOptions struct {
	ForceRecreate        bool
	WithRegistry         bool
	TargetHost           ssh.Host
	NoRecreate           bool
	Port                 string
	UseSSHControlSockets bool
}

func SupportsRegistry(noRegistry bool, targetHost ssh.Host) bool {
	return !noRegistry && !targetHost.IsPlainLocalhost()
}

func SupportsSSHControlSockets(goos string) bool {
	return goos != "windows"
}

func NewDeploymentStop(composeFile string, targetHost ssh.Host) operation.Sequence {
	ops := []operation.Operation{
		operation.NewDockerComposeStop(composeFile, targetHost),
	}
	return operation.NewSequence(ops...)
}

func NewDeployment(composeFile string, opts DeployOptions) (operation.Sequence, operation.Operation) {
	sourceHost := ssh.PlainLocalhost
	ops := []operation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}

	var cleanup operation.Operation
	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.WithRegistry {
			start, stop := ssh.NewSSHTunnel(opts.TargetHost, opts.Port, opts.UseSSHControlSockets)
			cleanup = stop
			ops = append(ops, operation.NewRunRegistry(opts.Port)...)
			ops = append(ops, start)
			ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, opts.TargetHost, opts.Port))
			ops = append(ops, stop)
		} else {
			ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, opts.TargetHost))
		}
	}
	upArgs := operation.DockerComposeUpArgs{
		ForceRecreate: opts.ForceRecreate,
		NoRecreate:    opts.NoRecreate,
	}
	ops = append(ops, operation.NewDockerComposeRun(composeFile, opts.TargetHost, upArgs))
	return operation.NewSequence(ops...), cleanup
}
