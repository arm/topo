package ssh

import (
	"bytes"
	"fmt"
	"os/exec"
	"slices"
)

// RunCommand executes a command on dest over SSH and returns stdout.
// stderr is classified into a typed error when a known failure pattern is detected.
// Pass stdin data as optional parameter, or nil for no stdin.
func RunCommand(dest Destination, command string, stdin []byte, sshArgs ...string) (string, error) {
	args := slices.Concat(sshArgs, []string{"--", dest.String(), command})
	// #nosec G204 -- command should be validated by callers
	cmd := exec.Command("ssh", args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		stderr := stderrBuf.String()
		if classified := ClassifyStderr(stderr); classified != nil {
			err = classified
		}
		return stdoutBuf.String() + stderr, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", dest, err, stderr)
	}
	return stdoutBuf.String(), nil
}
