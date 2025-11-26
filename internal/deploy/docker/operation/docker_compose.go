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
	cmdOutput   io.Writer
	composeFile string
	host        ssh.Host
	args        []string
}

func NewDockerCompose(description string, cmdOutput io.Writer, composeFile string, h ssh.Host, args []string) *DockerCompose {
	return &DockerCompose{
		description: description,
		cmdOutput:   cmdOutput,
		composeFile: composeFile,
		host:        h,
		args:        args,
	}
}

func (dc *DockerCompose) Description() string {
	return dc.description
}

func (dc *DockerCompose) Run() error {
	cmd := dc.buildCommand()
	cmd.Stdout = dc.cmdOutput
	cmd.Stderr = dc.cmdOutput
	return cmd.Run()
}

func (dc *DockerCompose) DryRun(w io.Writer) error {
	cmd := dc.buildCommand()
	fmt.Fprintln(w, command.String(cmd))
	return nil
}

func (dc *DockerCompose) buildCommand() *exec.Cmd {
	return command.DockerCompose(dc.host, dc.composeFile, dc.args...)
}

func NewBuild(cmdOutput io.Writer, composeFile string, h ssh.Host) *DockerCompose {
	return NewDockerCompose("Build images", cmdOutput, composeFile, h, []string{"build"})
}

func NewPull(cmdOutput io.Writer, composeFile string, h ssh.Host) *DockerCompose {
	return NewDockerCompose("Pull images", cmdOutput, composeFile, h, []string{"pull"})
}

func NewRun(cmdOutput io.Writer, composeFile string, h ssh.Host) *DockerCompose {
	args := []string{"up", "-d", "--no-build", "--pull", "never"}
	return NewDockerCompose("Start services", cmdOutput, composeFile, h, args)
}
