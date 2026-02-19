package command

import (
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

func Docker(h ssh.Host, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(h), args...)
	return exec.Command("docker", cmdArgs...)
}

func DockerCompose(h ssh.Host, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(hostToArgs(h), composeArgs...)
	return exec.Command("docker", cmdArgs...)
}

func String(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func hostToArgs(h ssh.Host) []string {
	if h.IsPlainLocalhost() {
		return nil
	}
	return []string{"-H", h.AsURI()}
}
