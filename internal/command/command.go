package command

import (
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

func Docker(h ssh.Host, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(h), args...)
	// #nosec G204 -- Callers validate the command args
	return exec.Command("docker", cmdArgs...)
}

func DockerCompose(h ssh.Host, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(hostToArgs(h), composeArgs...)
	// #nosec G204 -- Callers validate the command args
	return exec.Command("docker", cmdArgs...)
}

func SSHKeyGen(keyType string, keyPath string, targetHost string) *exec.Cmd {
	sshKeyGenArgs := []string{"-t", keyType, "-f", keyPath, "-C", targetHost}
	// #nosec G204 -- Args are controlled by topo
	return exec.Command("ssh-keygen", sshKeyGenArgs...)
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
