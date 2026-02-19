package docker

import (
	"github.com/arm/topo/internal/deploy/docker/operation"
	goperation "github.com/arm/topo/internal/deploy/operation"
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

func NewDeploymentStop(composeFile string, targetHost ssh.Host) goperation.Sequence {
	ops := []goperation.Operation{
		operation.NewDockerComposeStop(composeFile, targetHost),
	}
	return goperation.NewSequence(ops...)
}

func NewDeployment(composeFile string, opts DeployOptions) (goperation.Sequence, goperation.Operation) {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}

	var cleanup goperation.Operation
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
	return goperation.NewSequence(ops...), cleanup
}
