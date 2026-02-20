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

// Exec runs a command on the target host. If the target is localhost, it runs locally.
// Pass stdin data as optional parameter, or nil for no stdin.
func Exec(target Host, command string, stdin []byte, sshArgs ...string) (string, error) {
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
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		combined := stdoutBuf.String() + stderrBuf.String()
		return combined, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", string(target), err, stderrBuf.String())
	}
	return stdoutBuf.String(), nil
}
