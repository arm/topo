package runner

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

type SSH struct {
	dest ssh.Destination
}

func NewSSH(dest ssh.Destination) *SSH {
	return &SSH{dest: dest}
}

func (r *SSH) Run(ctx context.Context, program string, args ...string) (string, string, error) {
	return r.RunWithStdin(ctx, nil, program, args...)
}

func (r *SSH) RunShell(ctx context.Context, script string) (string, string, error) {
	return r.RunShellWithStdinAndSSHArgs(ctx, script, nil)
}

func (r *SSH) RunWithStdin(ctx context.Context, stdin []byte, program string, args ...string) (string, string, error) {
	return r.exec(ctx, commandLine(program, args...), stdin, nil)
}

func (r *SSH) RunWithArgs(ctx context.Context, program string, sshArgs ...string) (string, string, error) {
	return r.exec(ctx, commandLine(program), nil, sshArgs)
}

func (r *SSH) RunShellWithStdinAndSSHArgs(ctx context.Context, script string, stdin []byte, sshArgs ...string) (string, string, error) {
	return r.exec(ctx, "sh -c -- "+command.QuoteArg(script), stdin, sshArgs)
}

func (r *SSH) BinaryExists(ctx context.Context, bin string) error {
	cmd, err := command.BinaryLookupCommand(bin)
	if err != nil {
		return err
	}
	if _, _, err := r.exec(ctx, cmd, nil, nil); err != nil {
		if errors.Is(err, ssh.ErrSSH) || errors.Is(err, ErrTimeout) {
			return err
		}
		return fmt.Errorf("%q not found on remote target's $PATH", bin)
	}
	return nil
}

func (r *SSH) exec(ctx context.Context, cmdStr string, stdin []byte, extraSSHArgs []string) (string, string, error) {
	args := append(multiplexArgs(), extraSSHArgs...)
	stdout, stderr, err := ssh.RunCommand(ctx, r.dest, cmdStr, stdin, args...)
	if err == nil {
		return stdout, stderr, nil
	}
	if ctx.Err() != nil {
		return "", "", ErrTimeout
	}
	return stdout, stderr, &command.Error{
		Command: fmt.Sprintf("ssh %s %s", r.dest, cmdStr),
		Stderr:  stderr,
		Err:     err,
	}
}

func multiplexArgs() []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	return []string{"-o", "ControlMaster=auto", "-o", "ControlPersist=10s", "-o", "ControlPath=~/.ssh/topo-cm-%C"}
}
