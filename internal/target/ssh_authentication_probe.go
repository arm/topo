package target

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

var (
	ErrHostKeyUnknown = errors.New("SSH host key is not known")
	ErrHostKeyChanged = errors.New("SSH host key has changed")
	ErrAuthFailed     = errors.New("SSH authentication failed")
)

type SSHAuthenticationProbeOptions struct {
	AcceptNewHostKeys bool
}

func (s SSHAuthenticationProbeOptions) SSHArgs() []string {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
		"-o", "NumberOfPasswordPrompts=0",
	}
	if s.AcceptNewHostKeys {
		args = append(args, "-o", "StrictHostKeyChecking=accept-new")
	} else {
		args = append(args, "-o", "StrictHostKeyChecking=yes")
	}
	return args
}

// ProbeSSHAuthentication verifies SSH connectivity by attempting public key authentication.
func ProbeSSHAuthentication(ctx context.Context, r *runner.SSH, opts SSHAuthenticationProbeOptions) error {
	_, err := r.RunWithArgs(ctx, "true", opts.SSHArgs()...)
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ssh.ErrHostKeyChanged):
		return ErrHostKeyChanged
	case errors.Is(err, ssh.ErrHostKeyUnknown):
		return ErrHostKeyUnknown
	case errors.Is(err, ssh.ErrAuthFailed):
		return ErrAuthFailed
	default:
		return err
	}
}
