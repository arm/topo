package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

const commandStartMarker = "__TOPO_COMMAND_START__"

// RunCommand executes a command in the destination's login shell and returns
// stdout and stderr separately, excluding output emitted before the command starts.
// Stderr is classified into a typed error when a known failure pattern is detected.
func RunCommand(ctx context.Context, dest Destination, cmdStr string, stdin []byte, sshArgs ...string) (string, string, error) {
	args := slices.Concat(sshArgs, []string{"--", dest.String(), wrapInLoginShell(cmdStr)})
	// #nosec G204 -- command should be validated by callers
	cmd := exec.CommandContext(ctx, "ssh", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil && ctx.Err() != nil {
		return "", "", ctx.Err()
	}

	stdout := trimLoginShellOutput(stdoutBuf.String())
	stderr := trimLoginShellOutput(stderrBuf.String())
	if err != nil {
		if classified := ClassifyStderr(stderr); classified != nil {
			err = classified
		}
		return stdout, stderr, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", dest, err, stderr)
	}
	return stdout, stderr, nil
}

func wrapInLoginShell(cmd string) string {
	payload := fmt.Sprintf("printf '%s\\n'; printf '%s\\n' >&2; %s", commandStartMarker, commandStartMarker, cmd)
	escaped := shellEscapeForDoubleQuotes(payload)
	return fmt.Sprintf(`/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"%s\""`, escaped)
}

func trimLoginShellOutput(output string) string {
	_, commandOutput, found := strings.Cut(output, commandStartMarker+"\n")
	if !found {
		return output
	}
	return commandOutput
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
