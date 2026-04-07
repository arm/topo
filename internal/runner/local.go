package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/google/shlex"
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

func (r *Local) BinaryExists(bin string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%q not found in $PATH", bin)
	}
	return nil
}

func (r *Local) exec(ctx context.Context, cmdStr string, stdin []byte) (string, error) {
	args, err := shlex.Split(cmdStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse command: %w", err)
	}
	// #nosec G204 -- command should be validated by callers
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return "", ErrTimeout
		}
		stderr := stderrBuf.String()
		return stdoutBuf.String() + stderr, fmt.Errorf("local command failed: %w | stderr: %s", err, stderr)
	}
	return stdoutBuf.String(), nil
}
