package operation

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
)

type DockerComposePipeTransfer struct {
	composeFile string
	source      command.Host
	dest        command.Host
}

func NewDockerComposePipeTransfer(composeFile string, source, dest command.Host) *DockerComposePipeTransfer {
	return &DockerComposePipeTransfer{
		composeFile: composeFile,
		source:      source,
		dest:        dest,
	}
}

func (t *DockerComposePipeTransfer) Description() string {
	return "Transfer images"
}

func (t *DockerComposePipeTransfer) Run(output io.Writer) error {
	return docker.TransferImagesViaPipe(context.Background(), output, t.composeFile, t.source, t.dest)
}
