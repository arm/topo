package docker

import (
	"io"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/deploy/host"
	goperation "github.com/arm-debug/topo-cli/internal/deploy/operation"
)

func NewDeployment(cmdOutput io.Writer, composeFile string, targetHost host.Host) goperation.Sequence {
	sourceHost := host.Local
	return goperation.NewSequence(
		cmdOutput,
		operation.NewBuild(cmdOutput, composeFile, sourceHost),
		operation.NewPull(cmdOutput, composeFile, sourceHost),
		operation.NewTransfer(cmdOutput, composeFile, sourceHost, targetHost),
		operation.NewRun(cmdOutput, composeFile, targetHost),
	)
}
