package docker

import (
	"io"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/host"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
)

func NewDeployment(cmdOutput io.Writer, composeFile string, targetHost host.Host) operation.Sequence {
	sourceHost := host.Local
	return operation.Sequence{
		operation.NewBuild(cmdOutput, composeFile, sourceHost),
		operation.NewPull(cmdOutput, composeFile, sourceHost),
		operation.NewTransfer(cmdOutput, composeFile, sourceHost, targetHost),
		operation.NewRun(cmdOutput, composeFile, targetHost),
	}
}
