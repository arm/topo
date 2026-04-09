package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Local struct{}

func NewLocal() *Local {
	return &Local{}
}

func (r *Local) Run(ctx context.Context, cmdStr string) (string, error) {
	return r.exec(ctx, cmdStr, nil)
}

func (r *Local) RunWithStdin(ctx context.Context, cmdStr string, stdin []byte) (string, error) {
	return r.exec(ctx, cmdStr, stdin)
}

func (r *Local) exec(ctx context.Context, cmdStr string, stdin []byte) (string, error) {
	// #nosec G204 -- command should be validated by callers
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return "", ErrTimeout
		}
		stderr := stderrBuf.String()
		return stdoutBuf.String() + stderr, fmt.Errorf("local command failed: %w | stderr: %s", err, stderr)
	}
	return stdoutBuf.String(), nil
}
