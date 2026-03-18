package ssh

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

var BinaryRegex = regexp.MustCompile(`^[A-Za-z0-9_+-]+$`)

func ValidateBinaryName(bin string) error {
	if !BinaryRegex.MatchString(bin) {
		return fmt.Errorf("%q is not a valid binary name (contains invalid characters)", bin)
	}
	return nil
}

func shellEscapeForDoubleQuotes(s string) string {
	// Escape for TWO nested double-quoted shell layers- need three `\\\`.
	// /bin/sh -c "exec ${SHELL} -l -c \"<command>\""
	repl := strings.NewReplacer(
		`\`, `\\\\`,
		`"`, `\\\"`,
		`$`, `\\\$`,
		"`", `\\\`+"`",
	)
	return repl.Replace(s)
}

func ShellCommandStr(cmdStr string) string {
	escaped := shellEscapeForDoubleQuotes(cmdStr)
	return fmt.Sprintf(`/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"%s\""`, escaped)
}

// CommandFunc builds a command to be executed on a target host.
type CommandFunc func(target Host, cmdStr string, stdin []byte, sshArgs ...string) *exec.Cmd

// Command builds a command to be executed on the target host. If the target is localhost, it will run locally when executed.
// Pass stdin data as optional parameter, or nil for no stdin.
func Command(target Host, cmdStr string, stdin []byte, sshArgs ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if target.IsPlainLocalhost() {
		// #nosec G204 -- cmdStr should be validated by callers
		cmd = exec.Command("/bin/sh", "-c", cmdStr)
	} else {
		args := slices.Concat(sshArgs, []string{"--", string(target), cmdStr})
		// #nosec G204 -- cmdStr should be validated by callers
		cmd = exec.Command("ssh", args...)
	}

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd
}
