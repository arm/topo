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

func (r *SSH) Run(ctx context.Context, cmdStr string) (string, string, error) {
	return r.exec(ctx, cmdStr, nil, nil)
}

func (r *SSH) RunWithStdin(ctx context.Context, cmdStr string, stdin []byte) (string, string, error) {
	return r.exec(ctx, cmdStr, stdin, nil)
}

func (r *SSH) RunWithArgs(ctx context.Context, cmdStr string, args ...string) (string, string, error) {
	return r.exec(ctx, cmdStr, nil, args)
}

func (r *SSH) RunWithStdinAndArgs(ctx context.Context, cmdStr string, stdin []byte, args ...string) (string, string, error) {
	return r.exec(ctx, cmdStr, stdin, args)
}

func (r *SSH) BinaryExists(ctx context.Context, bin string) error {
	cmd, err := command.BinaryLookupCommand(bin)
	if err != nil {
		return err
	}
	if _, _, err := r.Run(ctx, cmd); err != nil {
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
	if err != nil && ctx.Err() != nil {
		return "", "", ErrTimeout
	}
	return stdout, stderr, err
}

func multiplexArgs() []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	return []string{"-o", "ControlMaster=auto", "-o", "ControlPersist=10s", "-o", "ControlPath=~/.ssh/topo-cm-%C"}
}
