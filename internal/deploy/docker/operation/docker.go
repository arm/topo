package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

type Docker struct {
	description string
	host        ssh.Host
	args        []string
}

func NewDocker(description string, h ssh.Host, args []string) *Docker {
	return &Docker{
		description: description,
		host:        h,
		args:        args,
	}
}

func (d *Docker) Description() string {
	return d.description
}

func (d *Docker) Run(cmdOutput io.Writer) error {
	cmd := d.buildCommand()
	cmd.Stdout = cmdOutput
	cmd.Stderr = cmdOutput
	return cmd.Run()
}

func (d *Docker) DryRun(output io.Writer) error {
	cmd := d.buildCommand()
	_, err := fmt.Fprintln(output, command.String(cmd))
	return err
}

func (d *Docker) buildCommand() *exec.Cmd {
	return command.Docker(d.host, d.args...)
}

func NewDockerPull(host ssh.Host, image string) *Docker {
	description := fmt.Sprintf("Pull image %s", image)
	return NewDocker(description, host, []string{"pull", image})
}

func NewDockerStart(host ssh.Host, container string) *Docker {
	description := fmt.Sprintf("Start container %s", container)
	return NewDocker(description, host, []string{"start", container})
}

func NewDockerRun(host ssh.Host, image string, container string, dockerArgs []string) *Docker {
	description := fmt.Sprintf("Run image %s as container %s", image, container)
	allArgs := []string{"run"}
	allArgs = append(allArgs, dockerArgs...)
	allArgs = append(allArgs, "--name", container)
	allArgs = append(allArgs, image)
	return NewDocker(description, host, allArgs)
}
