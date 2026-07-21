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
	// Suppress startup-file output, then restore both output streams before running cmd.
	return fmt.Sprintf(`/bin/sh -c "exec 3>&1 4>&2; exec ${SHELL:-/bin/sh} -l -c \"exec 1>&3 2>&4 3>&- 4>&-; %s\" >/dev/null 2>&1"`, escaped)
}

func BinaryLookupCommand(bin string) (string, error) {
	if err := ValidateBinaryName(bin); err != nil {
		return "", err
	}

	return fmt.Sprintf("command -v %s", bin), nil
}

func QuoteArg(s string) string {
	if s == "" {
		return "''"
	}

	if strings.ContainsAny(s, " \t\n\"'\\$`") {
		return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
	}

	return s
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
