package command

import (
	"fmt"
	"os/exec"
	"strings"
)

func SSHKeyGen(keyType string, keyPath string, targetHost string) *exec.Cmd {
	sshKeyGenArgs := []string{"-t", keyType, "-f", keyPath, "-C", targetHost}
	return exec.Command("ssh-keygen", sshKeyGenArgs...)
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
