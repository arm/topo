package target

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

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

type execSSH func(target ssh.Host, command string) (string, error)

type Connection struct {
	SSHTarget ssh.Host
	exec      execSSH
	opts      ConnectionOptions
}

type ConnectionOptions struct {
	AuthProbeEnabled  bool
	AcceptNewHostKeys bool
	AuthProbeInput    io.Reader
	AuthProbeOutput   io.Writer
}

var ErrPasswordAuthentication = errors.New("only password authentication is configured; key-based ssh is required")

func NewConnection(sshTarget string, exec execSSH, opts ConnectionOptions) Connection {
	return Connection{
		SSHTarget: ssh.Host(sshTarget),
		exec:      exec,
		opts:      opts,
	}
}

func (c *Connection) Run(command string) (string, error) {
	return c.exec(c.SSHTarget, command)
}

func (c *Connection) BinaryExists(bin string) (bool, error) {
	if err := ssh.ValidateBinaryName(bin); err != nil {
		return false, err
	}
	_, err := c.exec(c.SSHTarget, ssh.ShellCommand(fmt.Sprintf("command -v %s", bin)))
	return err == nil, nil
}

func (c *Connection) ProbeAuthentication() error {
	if !c.opts.AuthProbeEnabled {
		return nil
	}

	if !c.opts.AcceptNewHostKeys {
		err := c.runSSHProbe(knownHostProbeArgs)
		if err != nil && !errors.Is(err, ErrAuthenticationFailure) {
			return err
		}
	}

	isPwdAuth, err := c.isPasswordAuthenticated()
	if err != nil {
		return err
	}
	if isPwdAuth {
		return ErrPasswordAuthentication
	}
	return nil
}

var (
	ErrHostKeyVerification   = errors.New("ssh host key verification failed")
	ErrAuthenticationFailure = errors.New("ssh authentication failed")
)

func (c *Connection) isPasswordAuthenticated() (bool, error) {
	var extraArgs []string
	if c.opts.AcceptNewHostKeys {
		extraArgs = acceptNewHostKeyArgs
	}

	// If public key auth succeeds, the target doesn't require password auth.
	publicArgs := slices.Clone(publicKeyProbeArgs)
	if err := c.runSSHProbe(slices.Concat(publicArgs, extraArgs)); err == nil {
		return false, nil
	} else if !errors.Is(err, ErrAuthenticationFailure) {
		return false, err
	}

	// Public key was rejected. Check if the target accepts password auth.
	passwordArgs := slices.Clone(passwordProbeArgs)
	if err := c.runSSHProbe(slices.Concat(passwordArgs, extraArgs)); err == nil {
		return false, nil
	} else if errors.Is(err, ErrAuthenticationFailure) {
		return true, nil
	} else {
		return false, err
	}
}

func (c *Connection) runSSHProbe(sshArgs []string) error {
	stdout, stderr, err := ssh.Exec(c.SSHTarget, "true", nil, sshArgs...)
	if err == nil {
		return nil
	}
	output := stdout + stderr
	output = strings.ToLower(output)
	if strings.Contains(output, "host key verification failed") {
		return ErrHostKeyVerification
	}
	if strings.Contains(output, "permission denied") || strings.Contains(output, "authentication failed") || strings.Contains(output, "password") {
		return ErrAuthenticationFailure
	}
	return fmt.Errorf("ssh probe failed: %w", err)
}
