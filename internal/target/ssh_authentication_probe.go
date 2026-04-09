package target

import (
	"context"
	"errors"
	"strings"

	"github.com/arm/topo/internal/runner"
)

var (
	ErrHostKeyNew            = errors.New("ssh host key is not known")
	ErrHostKeyChanged        = errors.New("ssh host key has changed")
	ErrAuthenticationFailure = errors.New("ssh authentication failed")
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
// Returns ErrAuthenticationFailure if keys are not configured, or a host key
// error if the target's identity cannot be verified.
func (p SSHAuthenticationProbe) Probe(ctx context.Context) error {
	out, err := p.runner.RunWithArgs(ctx, "true", BuildProbeArgs(p.opts.AcceptNewHostKeys)...)
	return ClassifySSHOutput(out, err)
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

// ClassifySSHOutput maps raw SSH output to a domain error.
// Returns nil if err is nil, a domain error if the output is recognized,
// or the original error otherwise.
func ClassifySSHOutput(output string, err error) error {
	if err == nil {
		return nil
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, "host key verification failed") {
		if strings.Contains(lower, "has changed") {
			return ErrHostKeyChanged
		}
		return ErrHostKeyNew
	}
	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "authentication failed") || strings.Contains(lower, "password") {
		return ErrAuthenticationFailure
	}
	return err
}
