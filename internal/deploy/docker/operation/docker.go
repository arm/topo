package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

type Docker struct {
	description string
	host        ssh.Destination
	args        []string
}

func NewDocker(description string, h ssh.Destination, args []string) *Docker {
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

func (d *Docker) buildCommand() *exec.Cmd {
	return command.Docker(d.host, d.args...)
}

func NewDockerPull(host ssh.Destination, image string) *Docker {
	description := fmt.Sprintf("Pull image %s", image)
	return NewDocker(description, host, []string{"pull", image})
}

func NewDockerStart(host ssh.Destination, container string) *Docker {
	description := fmt.Sprintf("Start container %s", container)
	return NewDocker(description, host, []string{"start", container})
}

func NewDockerRun(host ssh.Destination, image string, container string, dockerArgs []string) *Docker {
	description := fmt.Sprintf("Run image %s as container %s", image, container)
	allArgs := []string{"run"}
	allArgs = append(allArgs, dockerArgs...)
	allArgs = append(allArgs, "--name", container)
	allArgs = append(allArgs, image)
	return NewDocker(description, host, allArgs)
}
