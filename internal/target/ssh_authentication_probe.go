package target

import (
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
	ErrHostKeyVerification    = errors.New("ssh host key verification failed")
	ErrAuthenticationFailure  = errors.New("ssh authentication failed")
)

// SSHRunner runs a command on a pre-configured SSH target with additional SSH client arguments.
type SSHRunner interface {
	RunWithArgs(command string, sshArgs ...string) (string, error)
}

type SSHAuthenticationProbeOptions struct {
	AcceptNewHostKeys bool
}

type SSHAuthenticationProbe struct {
	runner SSHRunner
	opts   SSHAuthenticationProbeOptions
}

func NewSSHAuthenticationProbe(runner SSHRunner, opts SSHAuthenticationProbeOptions) SSHAuthenticationProbe {
	return SSHAuthenticationProbe{runner: runner, opts: opts}
}

func (p SSHAuthenticationProbe) Probe() error {
	if !p.opts.AcceptNewHostKeys {
		err := p.runAuthenticationProbe(knownHostProbeArgs)
		if err != nil && !errors.Is(err, ErrAuthenticationFailure) {
			return err
		}
	}

	isPwdAuth, err := p.isPasswordAuthenticated()
	if err != nil {
		return err
	}
	if isPwdAuth {
		return ErrPasswordAuthentication
	}
	return nil
}

func (p SSHAuthenticationProbe) isPasswordAuthenticated() (bool, error) {
	var extraArgs []string
	if p.opts.AcceptNewHostKeys {
		extraArgs = acceptNewHostKeyArgs
	}

	// If public key auth succeeds, the target doesn't require password auth.
	publicArgs := slices.Clone(publicKeyProbeArgs)
	if err := p.runAuthenticationProbe(slices.Concat(publicArgs, extraArgs)); err == nil {
		return false, nil
	} else if !errors.Is(err, ErrAuthenticationFailure) {
		return false, err
	}

	// Public key was rejected. Check if the target accepts password auth.
	passwordArgs := slices.Clone(passwordProbeArgs)
	if err := p.runAuthenticationProbe(slices.Concat(passwordArgs, extraArgs)); err == nil {
		return false, nil
	} else if errors.Is(err, ErrAuthenticationFailure) {
		return true, nil
	} else {
		return false, err
	}
}

// All SSH authentication probes run the command "true" to check if the authentication method works.
// All sshArgs should be hardcoded SSH options, not user-provided arguments.
func (p SSHAuthenticationProbe) runAuthenticationProbe(sshArgs []string) error {
	out, err := p.runner.RunWithArgs("true", sshArgs...)
	if err == nil {
		return nil
	}
	output := strings.ToLower(out)
	if strings.Contains(output, "host key verification failed") {
		return ErrHostKeyVerification
	}
	if strings.Contains(output, "permission denied") || strings.Contains(output, "authentication failed") || strings.Contains(output, "password") {
		return ErrAuthenticationFailure
	}
	return err
}
