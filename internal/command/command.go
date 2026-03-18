package command

import (
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

func Docker(h ssh.SSHDestination, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(h), args...)
	return exec.Command("docker", cmdArgs...)
}

func DockerCompose(h ssh.SSHDestination, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(hostToArgs(h), composeArgs...)
	return exec.Command("docker", cmdArgs...)
}

func SSHKeyGen(keyType string, keyPath string, targetHost string) *exec.Cmd {
	sshKeyGenArgs := []string{"-t", keyType, "-f", keyPath, "-C", targetHost}
	return exec.Command("ssh-keygen", sshKeyGenArgs...)
}

func String(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func hostToArgs(h ssh.SSHDestination) []string {
	if h.IsPlainLocalhost() {
		return nil
	}
	return []string{"-H", h.AsURI()}
}
