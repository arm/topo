package target

import (
	"errors"
	"slices"
	"strings"

	"github.com/arm/topo/internal/runner"
	"github.com/arm/topo/internal/ssh"
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

type sshRunnerWithExtraArgs interface {
	RunWithArgs(command string, sshArgs ...string) (string, error)
}

// SSHProbeRunner implements sshRunnerWithExtraArgs by creating a fresh runner.SSH
// per call with the given sshArgs merged into SSHOptions.ExtraArgs.
type SSHProbeRunner struct {
	dest ssh.Destination
	opts runner.SSHOptions
}

func NewSSHProbeRunner(dest ssh.Destination, opts runner.SSHOptions) *SSHProbeRunner {
	return &SSHProbeRunner{dest: dest, opts: opts}
}

func (s *SSHProbeRunner) RunWithArgs(command string, sshArgs ...string) (string, error) {
	opts := s.opts
	opts.ExtraArgs = sshArgs
	r := runner.NewSSH(s.dest, opts)
	return r.Run(command)
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

func (p SSHAuthenticationProbe) Probe() error {
	if err := p.verifyKnownHost(); err != nil {
		return err
	}

	if err := p.authenticateUsingPublicKey(); err == nil {
		return nil
	} else if !errors.Is(err, ErrAuthenticationFailure) {
		return err
	}

	if err := p.authenticateUsingPassword(); err == nil {
		return nil
	} else if errors.Is(err, ErrAuthenticationFailure) {
		return ErrPasswordAuthentication
	} else {
		return err
	}
}

func (p SSHAuthenticationProbe) verifyKnownHost() error {
	if p.opts.AcceptNewHostKeys {
		return nil
	}

	err := p.runAuthenticationProbe(knownHostProbeArgs)
	if err == nil || errors.Is(err, ErrAuthenticationFailure) {
		return nil
	}

	return err
}

func (p SSHAuthenticationProbe) authenticateUsingPublicKey() error {
	publicKeyArgs := slices.Clone(publicKeyProbeArgs)
	if p.opts.AcceptNewHostKeys {
		publicKeyArgs = slices.Concat(publicKeyArgs, acceptNewHostKeyArgs)
	}

	return p.runAuthenticationProbe(publicKeyArgs)
}

func (p SSHAuthenticationProbe) authenticateUsingPassword() error {
	passwordArgs := slices.Clone(passwordProbeArgs)
	if p.opts.AcceptNewHostKeys {
		passwordArgs = slices.Concat(passwordArgs, acceptNewHostKeyArgs)
	}

	return p.runAuthenticationProbe(passwordArgs)
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
