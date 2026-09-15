package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"

	"github.com/arm/topo/internal/command"
)

// RunCommand executes a command in the destination's login shell and returns
// stdout and stderr separately, excluding output emitted before the command starts.
// Stderr is classified into a typed error when a known failure pattern is detected.
func RunCommand(ctx context.Context, dest Destination, cmdStr string, stdin []byte, sshArgs ...string) (string, string, error) {
	wrapper := command.NewLoginShellWrapper()
	args := slices.Concat(sshArgs, []string{"--", dest.String(), wrapper.Wrap(cmdStr)})
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

	stdout := wrapper.Unwrap(stdoutBuf.String())
	stderr := wrapper.Unwrap(stderrBuf.String())
	if err != nil {
		if classified := ClassifyStderr(stderr); classified != nil {
			err = classified
		}
		return stdout, stderr, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", dest, err, stderr)
	}
	return stdout, stderr, nil
}
