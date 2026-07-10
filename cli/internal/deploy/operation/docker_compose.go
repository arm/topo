package operation

import (
	"io"
	"os/exec"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/deploy/command"
)

type DockerCompose struct {
	description string
	composeFile string
	host        command.Host
	args        []string
}

func NewDockerCompose(description string, composeFile string, h command.Host, args []string) *DockerCompose {
	return &DockerCompose{
		description: description,
		composeFile: composeFile,
		host:        h,
		args:        args,
	}
}

func NewDockerComposeBuild(composeFile string, h command.Host) *DockerCompose {
	return NewDockerCompose("Build images", composeFile, h, []string{"build"})
}

type DockerComposePull struct {
	composeFile string
	host        command.Host
}

func NewDockerComposePull(composeFile string, h command.Host) *DockerComposePull {
	return &DockerComposePull{composeFile: composeFile, host: h}
}

func (p *DockerComposePull) Description() string { return "Pull images" }

func (p *DockerComposePull) Run(w io.Writer) error {
	services, err := compose.PullableServices(p.composeFile)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return nil
	}
	args := append([]string{"pull"}, services...)
	cmd := command.DockerCompose(p.host, p.composeFile, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	return cmd.Run()
}

func NewDockerComposeStop(composeFile string, h command.Host) *DockerCompose {
	return NewDockerCompose("Stop services", composeFile, h, []string{"stop"})
}

func NewDockerComposeUp(composeFile string, h command.Host, mode RecreateMode) *DockerCompose {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	switch mode {
	case RecreateModeForce:
		args = append(args, "--force-recreate")
	case RecreateModeNone:
		args = append(args, "--no-recreate")
	}
	return NewDockerCompose("Start services", composeFile, h, args)
}

func (dc *DockerCompose) Description() string {
	return dc.description
}

func (dc *DockerCompose) Run(cmdOutput io.Writer) error {
	cmd := dc.buildCommand()
	cmd.Stdout = cmdOutput
	cmd.Stderr = cmdOutput
	return cmd.Run()
}

func (dc *DockerCompose) buildCommand() *exec.Cmd {
	return command.DockerCompose(dc.host, dc.composeFile, dc.args...)
}

type RecreateMode int

const (
	RecreateModeDefault RecreateMode = iota
	RecreateModeForce
	RecreateModeNone
)
