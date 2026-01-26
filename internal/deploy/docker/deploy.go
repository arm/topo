package docker

import (
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

type DeployOptions struct {
	ForceRecreate bool
	WithRegistry  bool
	TargetHost    ssh.Host
	NoRecreate    bool
}

func SupportsRegistry(noRegistry bool, targetHost ssh.Host, goos string) bool {
	return !noRegistry && !targetHost.IsPlainLocalhost() && goos != "windows"
}

func NewDeploymentStop(composeFile string, targetHost ssh.Host) goperation.Sequence {
	ops := []goperation.Operation{
		operation.NewDockerComposeStop(composeFile, targetHost),
	}
	return goperation.NewSequence(ops...)
}

func NewDeployment(composeFile string, opts DeployOptions) goperation.Sequence {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}

	if !opts.TargetHost.IsPlainLocalhost() {
		if opts.WithRegistry {
			ops = append(ops, operation.NewRunRegistry()...)
			ops = append(ops, ssh.NewSSHTunnelStart(opts.TargetHost))
			ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, opts.TargetHost))
			ops = append(ops, ssh.NewSSHTunnelStop(opts.TargetHost))
		} else {
			ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, opts.TargetHost))
		}
	}
	upArgs := operation.DockerComposeUpArgs{
		ForceRecreate: opts.ForceRecreate,
		NoRecreate:    opts.NoRecreate,
	}
	ops = append(ops, operation.NewDockerComposeRun(composeFile, opts.TargetHost, upArgs))
	return goperation.NewSequence(ops...)
}
