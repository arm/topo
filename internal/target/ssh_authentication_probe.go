package target

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrPasswordAuthentication = errors.New("key-based SSH authentication is not setup")
	ErrHostKeyNew             = errors.New("ssh host key is not known")
	ErrHostKeyChanged         = errors.New("ssh host key has changed")
	ErrAuthenticationFailure  = errors.New("ssh authentication failed")
)

type sshRunnerWithExtraArgs interface {
	RunWithArgs(ctx context.Context, command string, sshArgs ...string) (string, error)
}

type SSHAuthenticationProbeOptions struct {
	AcceptNewHostKeys bool
}

func NewSSHAuthenticationProbe(r sshRunnerWithExtraArgs, opts SSHAuthenticationProbeOptions) SSHAuthenticationProbe {
	return SSHAuthenticationProbe{runner: r, opts: opts}
}

type SSHAuthenticationProbe struct {
	runner sshRunnerWithExtraArgs
	opts   SSHAuthenticationProbeOptions
}

func (p SSHAuthenticationProbe) Probe(ctx context.Context) error {
	if err := p.verifyKnownHost(ctx); err != nil {
		return err
	}

	if err := p.authenticateUsingPublicKey(ctx); err == nil {
		return nil
	} else if !errors.Is(err, ErrAuthenticationFailure) {
		return err
	}

	if err := p.authenticateUsingPassword(ctx); err == nil {
		return nil
	} else if errors.Is(err, ErrAuthenticationFailure) {
		return ErrPasswordAuthentication
	} else {
		return err
	}
}

func (p SSHAuthenticationProbe) verifyKnownHost(ctx context.Context) error {
	if p.opts.AcceptNewHostKeys {
		return nil
	}

	err := p.runAuthenticationProbe(ctx, BuildProbeArgs(KnownHost, false))
	if err == nil || errors.Is(err, ErrAuthenticationFailure) {
		return nil
	}

	return err
}

func (p SSHAuthenticationProbe) authenticateUsingPublicKey(ctx context.Context) error {
	return p.runAuthenticationProbe(ctx, BuildProbeArgs(PublicKey, p.opts.AcceptNewHostKeys))
}

func (p SSHAuthenticationProbe) authenticateUsingPassword(ctx context.Context) error {
	return p.runAuthenticationProbe(ctx, BuildProbeArgs(Password, p.opts.AcceptNewHostKeys))
}

// All SSH authentication probes run the command "true" to check if the authentication method works.
// All sshArgs should be hardcoded SSH options, not user-provided arguments.
func (p SSHAuthenticationProbe) runAuthenticationProbe(ctx context.Context, sshArgs []string) error {
	out, err := p.runner.RunWithArgs(ctx, "true", sshArgs...)
	return ClassifySSHOutput(out, err)
}

type ProbeMethod int

const (
	PublicKey ProbeMethod = iota
	Password
	KnownHost
)

// BuildProbeArgs returns the SSH arguments for a given probe method.
func BuildProbeArgs(method ProbeMethod, acceptNewHostKeys bool) []string {
	var args []string
	switch method {
	case PublicKey:
		args = []string{
			"-o", "BatchMode=yes",
			"-o", "PreferredAuthentications=publickey",
		}
	case Password:
		args = []string{
			"-o", "BatchMode=yes",
			"-o", "PreferredAuthentications=password",
			"-o", "NumberOfPasswordPrompts=0",
		}
	case KnownHost:
		args = []string{
			"-o", "StrictHostKeyChecking=yes",
			"-o", "PreferredAuthentications=publickey",
			"-o", "PasswordAuthentication=no",
			"-o", "NumberOfPasswordPrompts=0",
		}
	}
	if acceptNewHostKeys {
		args = append(args, "-o", "StrictHostKeyChecking=accept-new")
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
