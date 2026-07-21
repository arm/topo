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
