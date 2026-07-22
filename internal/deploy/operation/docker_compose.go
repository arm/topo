package operation

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
)

type DockerComposeBuild struct {
	composeFile string
	host        command.Host
}

func NewDockerComposeBuild(composeFile string, host command.Host) *DockerComposeBuild {
	return &DockerComposeBuild{composeFile: composeFile, host: host}
}

func (b *DockerComposeBuild) Description() string { return "Build images" }

func (b *DockerComposeBuild) Run(output io.Writer) error {
	return docker.BuildImages(context.Background(), output, b.composeFile, b.host)
}

type DockerComposePull struct {
	composeFile string
	host        command.Host
}

func NewDockerComposePull(composeFile string, host command.Host) *DockerComposePull {
	return &DockerComposePull{composeFile: composeFile, host: host}
}

func (p *DockerComposePull) Description() string { return "Pull images" }

func (p *DockerComposePull) Run(output io.Writer) error {
	return docker.PullImages(context.Background(), output, p.composeFile, p.host)
}

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

type DockerComposeUp struct {
	composeFile  string
	host         command.Host
	recreateMode docker.RecreateMode
}

func NewDockerComposeUp(composeFile string, host command.Host, mode docker.RecreateMode) *DockerComposeUp {
	return &DockerComposeUp{composeFile: composeFile, host: host, recreateMode: mode}
}

func (u *DockerComposeUp) Description() string { return "Start services" }

func (u *DockerComposeUp) Run(output io.Writer) error {
	return docker.StartServices(context.Background(), output, u.composeFile, u.host, u.recreateMode)
}
