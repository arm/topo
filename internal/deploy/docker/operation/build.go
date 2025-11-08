package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/host"
)

type Build struct {
	cmdOutput   io.Writer
	composeFile string
	host        host.Host
}

func NewBuild(cmdOutput io.Writer, composeFile string, h host.Host) *Build {
	return &Build{
		cmdOutput:   cmdOutput,
		composeFile: composeFile,
		host:        h,
	}
}

func (b *Build) buildCommand() *exec.Cmd {
	return command.DockerCompose(b.host, b.composeFile, "build")
}

func (b *Build) Run() error {
	cmd := b.buildCommand()
	cmd.Stdout = b.cmdOutput
	cmd.Stderr = b.cmdOutput
	return cmd.Run()
}

func (b *Build) DryRun(w io.Writer) error {
	cmd := b.buildCommand()
	fmt.Fprintln(w, command.String(cmd))
	return nil
}
