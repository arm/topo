package command

import (
	"fmt"
	"os/exec"
	"strings"
)

func Docker(host Host, args ...string) *exec.Cmd {
	cmdArgs := append(hostToArgs(host), args...)
	return exec.Command("docker", cmdArgs...)
}

func DockerCompose(host Host, composeFile string, args ...string) *exec.Cmd {
	composeArgs := append([]string{"compose", "-f", composeFile}, args...)
	cmdArgs := append(hostToArgs(host), composeArgs...)
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

func hostToArgs(h Host) []string {
	if h.value == "" {
		return nil
	}
	return []string{"-H", h.value}
}
