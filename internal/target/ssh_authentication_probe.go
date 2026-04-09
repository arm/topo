package target

import (
	"context"
	"errors"
	"slices"
	"strings"
)

var (
	publicKeyProbeArgs = []string{
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=publickey",
	}
	passwordProbeArgs = []string{
		"-o", "BatchMode=yes",
		"-o", "PreferredAuthentications=password",
		"-o", "NumberOfPasswordPrompts=0",
	}
	knownHostProbeArgs = []string{
		"-o", "StrictHostKeyChecking=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "PasswordAuthentication=no",
		"-o", "NumberOfPasswordPrompts=0",
	}
	acceptNewHostKeyArgs = []string{
		"-o", "StrictHostKeyChecking=accept-new",
	}
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

type SSHAuthenticationProbe struct {
	runner sshRunnerWithExtraArgs
	opts   SSHAuthenticationProbeOptions
}

func NewSSHAuthenticationProbe(r sshRunnerWithExtraArgs, opts SSHAuthenticationProbeOptions) SSHAuthenticationProbe {
	return SSHAuthenticationProbe{runner: r, opts: opts}
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

	err := p.runAuthenticationProbe(ctx, knownHostProbeArgs)
	if err == nil || errors.Is(err, ErrAuthenticationFailure) {
		return nil
	}

	return err
}

func (p SSHAuthenticationProbe) authenticateUsingPublicKey(ctx context.Context) error {
	publicKeyArgs := slices.Clone(publicKeyProbeArgs)
	if p.opts.AcceptNewHostKeys {
		publicKeyArgs = slices.Concat(publicKeyArgs, acceptNewHostKeyArgs)
	}

	return p.runAuthenticationProbe(ctx, publicKeyArgs)
}

func (p SSHAuthenticationProbe) authenticateUsingPassword(ctx context.Context) error {
	passwordArgs := slices.Clone(passwordProbeArgs)
	if p.opts.AcceptNewHostKeys {
		passwordArgs = slices.Concat(passwordArgs, acceptNewHostKeyArgs)
	}

	return p.runAuthenticationProbe(ctx, passwordArgs)
}

// All SSH authentication probes run the command "true" to check if the authentication method works.
// All sshArgs should be hardcoded SSH options, not user-provided arguments.
func (p SSHAuthenticationProbe) runAuthenticationProbe(ctx context.Context, sshArgs []string) error {
	out, err := p.runner.RunWithArgs(ctx, "true", sshArgs...)
	if err == nil {
		return nil
	}
	output := strings.ToLower(out)
	if strings.Contains(output, "host key verification failed") {
		if strings.Contains(output, "has changed") {
			return ErrHostKeyChanged
		}
		return ErrHostKeyNew
	}
	if strings.Contains(output, "permission denied") || strings.Contains(output, "authentication failed") || strings.Contains(output, "password") {
		return ErrAuthenticationFailure
	}
	return err
}
