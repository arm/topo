package runner

import (
	"context"
	"fmt"
	"runtime"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

type SSHOptions struct {
	Multiplex bool
}

func (opts SSHOptions) SSHArgs() []string {
	var args []string
	if opts.Multiplex && runtime.GOOS != "windows" {
		args = append(args, "-o", "ControlMaster=auto", "-o", "ControlPersist=10s", "-o", "ControlPath=~/.ssh/topo-cm-%r@%h:%p")
	}
	return args
}

type SSH struct {
	dest ssh.Destination
	opts SSHOptions
}

func NewSSH(dest ssh.Destination, opts SSHOptions) *SSH {
	return &SSH{dest: dest, opts: opts}
}

func (r *SSH) Run(ctx context.Context, cmdStr string) (string, error) {
	return r.exec(ctx, cmdStr, nil, nil)
}

func (r *SSH) RunWithStdin(ctx context.Context, cmdStr string, stdin []byte) (string, error) {
	return r.exec(ctx, cmdStr, stdin, nil)
}

func (r *SSH) RunWithArgs(ctx context.Context, cmdStr string, args ...string) (string, error) {
	args = append(args, r.opts.SSHArgs()...)
	out, err := ssh.RunCommand(ctx, r.dest, cmdStr, nil, args...)
	if err != nil && ctx.Err() != nil {
		return "", ErrTimeout
	}
	return out, err
}

func (r *SSH) RunWithStdinAndArgs(ctx context.Context, cmdStr string, stdin []byte, args ...string) (string, error) {
	args = append(args, r.opts.SSHArgs()...)
	out, err := ssh.RunCommand(ctx, r.dest, cmdStr, stdin, args...)
	if err != nil && ctx.Err() != nil {
		return "", ErrTimeout
	}
	return out, err
}

func (r *SSH) BinaryExists(ctx context.Context, bin string) error {
	cmd, err := command.BinaryLookupCommand(bin)
	if err != nil {
		return err
	}
	if _, err := r.Run(ctx, cmd); err != nil {
		return fmt.Errorf("%q not found on remote target's $PATH", bin)
	}
	return nil
}

func (r *SSH) exec(ctx context.Context, cmdStr string, stdin []byte, extraSSHArgs []string) (string, error) {
	out, err := ssh.RunCommand(ctx, r.dest, command.WrapInLoginShell(cmdStr), stdin, extraSSHArgs...)
	if err != nil && ctx.Err() != nil {
		return "", ErrTimeout
	}
	return out, err
}
