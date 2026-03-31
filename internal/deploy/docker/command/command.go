package command

import (
	"os/exec"
	"strings"
)

func Docker(host Host, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(host), args...)
	return exec.Command("docker", cmdArgs...)
}

func String(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func DockerCompose(host Host, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(hostToArgs(host), composeArgs...)
	return exec.Command("docker", cmdArgs...)
}

func hostToArgs(h Host) []string {
	if h.value == "" {
		return nil
	}
	return []string{"-H", h.value}
}
