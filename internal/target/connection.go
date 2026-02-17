package target

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/arm-debug/topo-cli/internal/ssh"
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

var ErrPasswordAuthenticationRequired = errors.New("password authentication required")

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
		if err := c.ensureKnownHost(c.opts.AuthProbeInput, c.opts.AuthProbeOutput); err != nil {
			return err
		}
	}

	needsSetup, err := c.isPasswordAuthenticated()
	if err != nil {
		return err
	}
	if needsSetup {
		return ErrPasswordAuthenticationRequired
	}
	return nil
}

var ErrHostKeyVerification = errors.New("ssh host key verification failed")

func (c *Connection) isPasswordAuthenticated() (bool, error) {
	publicArgs := append([]string{}, publicKeyProbeArgs...)
	if c.opts.AcceptNewHostKeys {
		publicArgs = append(publicArgs, acceptNewHostKeyArgs...)
	}
	publicOut, publicErr := c.runSSHProbe(publicArgs)
	if publicErr == nil {
		return false, nil
	}
	if isHostKeyVerificationFailure(publicOut) {
		return false, ErrHostKeyVerification
	}

	passwordArgs := append([]string{}, passwordProbeArgs...)
	if c.opts.AcceptNewHostKeys {
		passwordArgs = append(passwordArgs, acceptNewHostKeyArgs...)
	}
	passwordOut, passwordErr := c.runSSHProbe(passwordArgs)
	if passwordErr == nil {
		return false, nil
	}
	if isHostKeyVerificationFailure(passwordOut) {
		return false, ErrHostKeyVerification
	}

	if isAuthenticationFailure(passwordOut) {
		return true, nil
	}

	return false, fmt.Errorf("ssh probe failed: %w", passwordErr)
}

func (c *Connection) ensureKnownHost(in io.Reader, out io.Writer) error {
	args := append([]string{}, knownHostProbeArgs...)
	args = append(args, string(c.SSHTarget), "true")
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = in
	var buf bytes.Buffer
	writer := io.Writer(&buf)
	if out != nil {
		writer = io.MultiWriter(out, &buf)
	}
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Run(); err != nil {
		output := buf.String()
		switch {
		case isHostKeyVerificationFailure(output):
			return ErrHostKeyVerification
		case isAuthenticationFailure(output):
			return nil
		default:
			return err
		}
	}
	return nil
}

func (c *Connection) runSSHProbe(sshArgs []string) (string, error) {
	stdout, stderr, err := ssh.Exec(c.SSHTarget, "true", nil, sshArgs...)
	return stdout + stderr, err
}

func isHostKeyVerificationFailure(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "host key verification failed")
}

func isAuthenticationFailure(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "permission denied") || strings.Contains(lower, "authentication failed") || strings.Contains(lower, "password")
}
