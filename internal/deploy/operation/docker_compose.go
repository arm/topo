package operation

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
)

type DockerComposeStop struct {
	composeFile string
	host        command.Host
}

func NewDockerComposeStop(composeFile string, host command.Host) *DockerComposeStop {
	return &DockerComposeStop{composeFile: composeFile, host: host}
}

func (s *DockerComposeStop) Description() string { return "Stop services" }

func (s *DockerComposeStop) Run(output io.Writer) error {
	return docker.StopServices(context.Background(), output, s.composeFile, s.host)
}
