package operation

import (
	"fmt"
	"io"
	"os/exec"

	dockercommand "github.com/arm/topo/internal/deploy/docker/docker_command"
)

type Docker struct {
	description string
	host        dockercommand.Host
	args        []string
}

func NewDocker(description string, h dockercommand.Host, args []string) *Docker {
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
	return dockercommand.Docker(d.host, d.args...)
}

func NewDockerPull(host dockercommand.Host, image string) *Docker {
	description := fmt.Sprintf("Pull image %s", image)
	return NewDocker(description, host, []string{"pull", image})
}

func NewDockerStart(host dockercommand.Host, container string) *Docker {
	description := fmt.Sprintf("Start container %s", container)
	return NewDocker(description, host, []string{"start", container})
}

func NewDockerRun(host dockercommand.Host, image string, container string, dockerArgs []string) *Docker {
	description := fmt.Sprintf("Run image %s as container %s", image, container)
	allArgs := []string{"run"}
	allArgs = append(allArgs, dockerArgs...)
	allArgs = append(allArgs, "--name", container)
	allArgs = append(allArgs, image)
	return NewDocker(description, host, allArgs)
}
