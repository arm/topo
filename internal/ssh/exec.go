package ssh

import (
	"bytes"
	"os/exec"
	"slices"
)

// ExecCmd builds a command to be executed on the target host. If the target is localhost, it will run locally when executed.
// Pass stdin data as optional parameter, or nil for no stdin.
func ExecCmd(target Destination, command string, stdin []byte, sshArgs ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if target.IsPlainLocalhost() {
		// #nosec G204 -- command should be validated by callers
		cmd = exec.Command("/bin/sh", "-c", command)
	} else {
		args := slices.Concat(sshArgs, []string{"--", target.String(), command})
		// #nosec G204 -- command should be validated by callers
		cmd = exec.Command("ssh", args...)
	}

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd
}
