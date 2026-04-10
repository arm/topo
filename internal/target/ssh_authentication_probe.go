package target

import (
	"context"
	"errors"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
)

var (
	ErrHostKeyUnknown = errors.New("ssh host key is not known")
	ErrHostKeyChanged = errors.New("ssh host key has changed")
	ErrAuthFailed     = errors.New("ssh authentication failed")
)

type SSHAuthenticationProbeOptions struct {
	AcceptNewHostKeys bool
}

type SSHAuthenticationProbe struct {
	runner *runner.SSH
	opts   SSHAuthenticationProbeOptions
}

func NewSSHAuthenticationProbe(r *runner.SSH, opts SSHAuthenticationProbeOptions) SSHAuthenticationProbe {
	return SSHAuthenticationProbe{runner: r, opts: opts}
}

// Probe verifies SSH connectivity by attempting public key authentication.
func (p SSHAuthenticationProbe) Probe(ctx context.Context) error {
	_, err := p.runner.RunWithArgs(ctx, "true", BuildProbeArgs(p.opts.AcceptNewHostKeys)...)
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

// BuildProbeArgs returns SSH arguments for a public key authentication probe.
func BuildProbeArgs(acceptNewHostKeys bool) []string {
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
