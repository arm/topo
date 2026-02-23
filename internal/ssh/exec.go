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

func ShellCommand(command string) string {
	escaped := shellEscapeForDoubleQuotes(command)
	return fmt.Sprintf(`/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"%s\""`, escaped)
}

// ExecCmd builds a command to be executed on the target host. If the target is localhost, it will run locally when executed.
// Pass stdin data as optional parameter, or nil for no stdin.
func ExecCmd(target Host, command string, stdin []byte, sshArgs ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if target.IsPlainLocalhost() {
		cmd = exec.Command("/bin/sh", "-c", command)
	} else {
		args := slices.Concat(sshArgs, []string{"--", string(target), command})
		cmd = exec.Command("ssh", args...)
	}

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd
}
