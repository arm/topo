package runner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/arm/topo/internal/command"
)

type Local struct{}

func NewLocal() *Local {
	return &Local{}
}

func (r *Local) Run(ctx context.Context, program string, args ...string) (string, string, error) {
	return r.exec(ctx, nil, program, args...)
}

func (r *Local) RunShell(ctx context.Context, script string) (string, string, error) {
	if err := r.BinaryExists(ctx, "sh"); err != nil {
		return "", "", err
	}
	return r.exec(ctx, nil, "sh", "-c", "--", script)
}

func (r *Local) RunWithStdin(ctx context.Context, stdin []byte, program string, args ...string) (string, string, error) {
	return r.exec(ctx, stdin, program, args...)
}

func (r *Local) BinaryExists(_ context.Context, bin string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%q not found in $PATH", bin)
	}
	return nil
}

func (r *Local) exec(ctx context.Context, stdin []byte, program string, args ...string) (string, string, error) {
	// #nosec G204 -- command should be validated by callers
	cmd := exec.CommandContext(ctx, program, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", "", ErrTimeout
		}
		stderr := stderrBuf.String()
		return stdoutBuf.String(), stderr, &command.Error{
			Command: commandLine(program, args...),
			Stderr:  stderr,
			Err:     err,
		}
	}
	return stdoutBuf.String(), stderrBuf.String(), nil
}
