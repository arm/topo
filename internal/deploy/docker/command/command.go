package command

import (
	"os/exec"
	"strings"

	"github.com/arm-debug/topo-cli/internal/deploy/host"
)

func Docker(h host.Host, args ...string) *exec.Cmd {
	cmdArgs := append(h.DockerCommandArgs(), args...)
	return exec.Command("docker", cmdArgs...)
}

func DockerCompose(h host.Host, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(h.DockerCommandArgs(), composeArgs...)
	return exec.Command("docker", cmdArgs...)
}

func String(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}
