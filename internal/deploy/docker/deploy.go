package docker

import (
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

func NewDeployment(composeFile string, targetHost ssh.Host) goperation.Sequence {
	sourceHost := ssh.Empty
	return goperation.NewSequence(
		operation.NewBuild(composeFile, sourceHost),
		operation.NewPull(composeFile, sourceHost),
		operation.NewTransfer(composeFile, sourceHost, targetHost),
		operation.NewRun(composeFile, targetHost),
	)
}
