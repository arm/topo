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

func ExecSSH(target Host, command string, sshArgs ...string) (string, error) {
	cmd := exec.Command("ssh", slices.Concat(sshArgs, []string{string(target), command})...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		combined := stdout.String() + stderr.String()
		return combined, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", target, err, stderr.String())
	}

	return stdout.String(), nil
}

// Exec runs a command on the target host. If the target is localhost, it runs locally.
// Pass stdin data as optional parameter, or nil for no stdin.
func Exec(target Host, command string, stdin []byte, sshArgs ...string) (stdout, stderr string, err error) {
	var cmd *exec.Cmd
	if target.IsPlainLocalhost() {
		cmd = exec.Command("/bin/sh", "-c", command)
	} else {
		args := append([]string{}, sshArgs...)
		args = append(args, string(target), command)
		cmd = exec.Command("ssh", args...)
	}

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// ExecWithShell runs a command in a login shell on the target host. If the target is localhost, it runs locally.
func ExecWithShell(target Host, command string) (string, error) {
	stdout, stderr, err := Exec(target, ShellCommand(command), nil)
	if err != nil {
		return "", fmt.Errorf("command failed: %w | stderr: %s", err, stderr)
	}
	return stdout, nil
}
