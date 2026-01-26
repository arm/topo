package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

type DockerCompose struct {
	description string
	composeFile string
	host        ssh.Host
	args        []string
}

type DockerComposeUpArgs struct {
	ForceRecreate bool
	NoRecreate    bool
}

func NewDockerCompose(description string, composeFile string, h ssh.Host, args []string) *DockerCompose {
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

func (dc *DockerCompose) DryRun(output io.Writer) error {
	cmd := dc.buildCommand()
	_, err := fmt.Fprintln(output, command.String(cmd))
	return err
}

func (dc *DockerCompose) buildCommand() *exec.Cmd {
	return command.DockerCompose(dc.host, dc.composeFile, dc.args...)
}

func NewDockerComposeBuild(composeFile string, h ssh.Host) *DockerCompose {
	return NewDockerCompose("Build images", composeFile, h, []string{"build"})
}

func NewDockerComposePull(composeFile string, h ssh.Host) *DockerCompose {
	return NewDockerCompose("Pull images", composeFile, h, []string{"pull"})
}

func NewDockerComposeStop(composeFile string, h ssh.Host) *DockerCompose {
	return NewDockerCompose("Stop services", composeFile, h, []string{"stop"})
}

func NewDockerComposeRun(composeFile string, h ssh.Host, upArgs DockerComposeUpArgs) *DockerCompose {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	if upArgs.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	if upArgs.NoRecreate {
		args = append(args, "--no-recreate")
	}
	return NewDockerCompose("Start services", composeFile, h, args)
}
