package probe

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

var (
	ErrHostKeyUnknown = errors.New("ssh host key is unknown")
	ErrHostKeyChanged = errors.New("ssh host key has changed")
	ErrAuthFailed     = errors.New("ssh authentication failed")
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
