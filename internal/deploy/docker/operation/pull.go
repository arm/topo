package operation

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/host"
)

type Pull struct {
	cmdOutput   io.Writer
	composeFile string
	host        host.Host
}

func NewPull(cmdOutput io.Writer, composeFile string, h host.Host) *Pull {
	return &Pull{
		cmdOutput:   cmdOutput,
		composeFile: composeFile,
		host:        h,
	}
}

func (p *Pull) buildCommand() *exec.Cmd {
	return command.DockerCompose(p.host, p.composeFile, "pull")
}

func (p *Pull) Run() error {
	cmd := p.buildCommand()
	cmd.Stdout = p.cmdOutput
	cmd.Stderr = p.cmdOutput
	return cmd.Run()
}

func (p *Pull) DryRun(w io.Writer) error {
	cmd := p.buildCommand()
	fmt.Fprintln(w, command.String(cmd))
	return nil
}
