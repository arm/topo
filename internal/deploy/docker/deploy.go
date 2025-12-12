package docker

import (
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

func NewDeployment(composeFile string, targetHost ssh.Host) goperation.Sequence {
	sourceHost := ssh.PlainLocalhost
	ops := []goperation.Operation{
		operation.NewDockerComposeBuild(composeFile, sourceHost),
		operation.NewDockerComposePull(composeFile, sourceHost),
	}
	if !targetHost.IsPlainLocalhost() {
		ops = append(ops, operation.NewDockerComposePipeTransfer(composeFile, sourceHost, targetHost))
	}
	ops = append(ops, operation.NewDockerComposeRun(composeFile, targetHost))
	return goperation.NewSequence(ops...)
}
