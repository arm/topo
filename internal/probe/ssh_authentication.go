package probe

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

var (
	ErrHostKeyUnknown   = errors.New("SSH host key is not known")
	ErrHostKeyChanged   = errors.New("SSH host key has changed")
	ErrAuthFailed       = errors.New("SSH authentication failed")
	ErrTooManyAuthFails = errors.New("SSH too many authentication failures")
)

// SSHAuthentication verifies SSH connectivity by attempting public key authentication.
func SSHAuthentication(ctx context.Context, r *runner.SSH, acceptNewHostKeys bool) error {
	_, err := r.RunWithArgs(ctx, "true", sshAuthArgs(acceptNewHostKeys)...)
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
	case errors.Is(err, ssh.ErrTooManyAuthFails):
		return ErrTooManyAuthFails
	default:
		return err
	}
}

func sshAuthArgs(acceptNewHostKeys bool) []string {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
		"-o", "NumberOfPasswordPrompts=0",
	}
	if acceptNewHostKeys {
		args = append(args, "-o", "StrictHostKeyChecking=accept-new")
	} else {
		args = append(args, "-o", "StrictHostKeyChecking=yes")
	}
	return args
}
