package docker

import (
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

func SupportsRegistry(noRegistry bool, targetHost ssh.Host, goos string) bool {
	return !noRegistry && !targetHost.IsPlainLocalhost() && goos != "windows"
}

func NewDeploymentStop(composeFile string, targetHost ssh.Host) goperation.Sequence {
	ops := []goperation.Operation{
		operation.NewDockerComposeStop(composeFile, targetHost),
	}
	return goperation.NewSequence(ops...)
}

func NewDeployment(composeFile string, targetHost ssh.Host, forceRecreate bool) goperation.Sequence {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}
	if !targetHost.IsPlainLocalhost() {
		ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, targetHost))
	}
	ops = append(ops, operation.NewDockerComposeRun(composeFile, targetHost, forceRecreate))
	return goperation.NewSequence(ops...)
}

func NewDeploymentWithRegistry(composeFile string, targetHost ssh.Host, forceRecreate bool) goperation.Sequence {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}
	if !targetHost.IsPlainLocalhost() {
		ops = append(ops, operation.NewRunRegistry()...)
		ops = append(ops, ssh.NewSSHTunnelStart(targetHost))
		ops = append(ops, operation.NewRegistryTransfer(composeFile, sourceHost, targetHost))
		ops = append(ops, ssh.NewSSHTunnelStop(targetHost))
	}
	ops = append(ops, operation.NewDockerComposeRun(composeFile, targetHost, forceRecreate))
	return goperation.NewSequence(ops...)
}
