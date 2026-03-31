package runner

import (
	"bytes"
	"fmt"
	"os/exec"
)

type Local struct{}

func NewLocal() Local {
	return Local{}
}

func (r *Local) Run(cmdStr string) (string, error) {
	return r.exec(cmdStr, nil)
}

func (r *Local) RunWithStdin(cmdStr string, stdin []byte) (string, error) {
	return r.exec(cmdStr, stdin)
}

func (r *Local) exec(cmdStr string, stdin []byte) (string, error) {
	// #nosec G204 -- command should be validated by callers
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		stderr := stderrBuf.String()
		return stdoutBuf.String() + stderr, fmt.Errorf("local command failed: %w | stderr: %s", err, stderr)
	}
	return stdoutBuf.String(), nil
}
