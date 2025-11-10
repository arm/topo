package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/host"
)

type Run struct {
	cmdOutput   io.Writer
	composeFile string
	host        host.Host
}

func NewRun(cmdOutput io.Writer, composeFile string, h host.Host) *Run {
	return &Run{
		cmdOutput:   cmdOutput,
		composeFile: composeFile,
		host:        h,
	}
}

func (r *Run) buildCommand() *exec.Cmd {
	return command.DockerCompose(r.host, r.composeFile,
		"up",
		"-d",
		"--no-build",
		"--pull", "never",
	)
}

func (r *Run) Run() error {
	cmd := r.buildCommand()
	cmd.Stdout = r.cmdOutput
	cmd.Stderr = r.cmdOutput
	return cmd.Run()
}

func (r *Run) DryRun(w io.Writer) error {
	cmd := r.buildCommand()
	fmt.Fprintln(w, command.String(cmd))
	return nil
}
