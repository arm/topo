package docker

import (
	"io"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

func NewDeployment(cmdOutput io.Writer, composeFile string, targetHost ssh.Host) goperation.Sequence {
	sourceHost := ssh.Empty
	return goperation.NewSequence(
		cmdOutput,
		operation.NewBuild(cmdOutput, composeFile, sourceHost),
		operation.NewPull(cmdOutput, composeFile, sourceHost),
		operation.NewTransfer(cmdOutput, composeFile, sourceHost, targetHost),
		operation.NewRun(cmdOutput, composeFile, targetHost),
	)
}
