package command

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/arm/topo/internal/ssh"
)

func Docker(h ssh.Destination, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(h), args...)
	return exec.Command("docker", cmdArgs...)
}

func DockerCompose(h ssh.Destination, composeFile string, args ...string) *exec.Cmd {
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

func WrapInLoginShell(cmd string) string {
	escaped := shellEscapeForDoubleQuotes(cmd)
	return fmt.Sprintf(`/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"%s\""`, escaped)
}

func UnsafeBinaryLookupCommand(bin string) string {
	return WrapInLoginShell(fmt.Sprintf("command -v %s", bin))
}

func BinaryLookupCommand(bin string) (string, error) {
	if err := ValidateBinaryName(bin); err != nil {
		return "", err
	}

	return UnsafeBinaryLookupCommand(bin), nil
}

func shellEscapeForDoubleQuotes(s string) string {
	repl := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\\\"`,
		`$`, `\\\$`,
		"`", `\\\`+"`",
	)
	return repl.Replace(s)
}

func hostToArgs(h ssh.Destination) []string {
	if h.IsPlainLocalhost() {
		return nil
	}
	return []string{"-H", h.AsURI()}
}
