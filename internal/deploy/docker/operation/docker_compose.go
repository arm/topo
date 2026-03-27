package operation

import (
	"io"
	"os/exec"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

type DockerCompose struct {
	description string
	composeFile string
	host        ssh.Destination
	args        []string
}

func NewDockerCompose(description string, composeFile string, h ssh.Destination, args []string) *DockerCompose {
	return &DockerCompose{
		description: description,
		composeFile: composeFile,
		host:        h,
		args:        args,
	}
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

func NewDockerComposeBuild(composeFile string, h ssh.Destination) *DockerCompose {
	return NewDockerCompose("Build images", composeFile, h, []string{"build"})
}

func NewDockerComposePull(composeFile string, h ssh.Destination) *DockerCompose {
	return NewDockerCompose("Pull images", composeFile, h, []string{"pull"})
}

func NewDockerComposeStop(composeFile string, h ssh.Destination) *DockerCompose {
	return NewDockerCompose("Stop services", composeFile, h, []string{"stop"})
}

type RecreateMode int

const (
	RecreateModeDefault RecreateMode = iota
	RecreateModeForce
	RecreateModeNone
)

func NewDockerComposeUp(composeFile string, h ssh.Destination, mode RecreateMode) *DockerCompose {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	switch mode {
	case RecreateModeForce:
		args = append(args, "--force-recreate")
	case RecreateModeNone:
		args = append(args, "--no-recreate")
	}
	return NewDockerCompose("Start services", composeFile, h, args)
}
