package command

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var unquotedShellArgPattern = regexp.MustCompile(`^[[:alnum:]_.-]+$`)

func SSHKeyGen(keyType string, keyPath string, targetHost string) *exec.Cmd {
	sshKeyGenArgs := []string{"-t", keyType, "-f", keyPath, "-C", targetHost}
	return exec.Command("ssh-keygen", sshKeyGenArgs...)
}

func BinaryLookupCommand(bin string) (string, error) {
	if err := ValidateBinaryName(bin); err != nil {
		return "", err
	}

	return fmt.Sprintf("command -v %s", bin), nil
}

func QuoteArg(s string) string {
	if unquotedShellArgPattern.MatchString(s) {
		return s
	}

	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
