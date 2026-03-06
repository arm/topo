package target

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	commandpkg "github.com/arm/topo/internal/command"
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

type ExecSSH func(target ssh.Host, command string, stdin []byte, sshArgs ...string) *exec.Cmd

type Connection struct {
	SSHTarget ssh.Host
	exec      ExecSSH
	opts      ConnectionOptions
}

type ConnectionOptions struct {
	AcceptNewHostKeys bool
	AuthProbeInput    io.Reader
	AuthProbeOutput   io.Writer
	WithLoginShell    bool
	WithStdin         []byte
	Multiplex         bool
	WithMockExec      ExecSSH
}

var ErrPasswordAuthentication = errors.New("only password authentication is configured; key-based ssh is required")

func NewConnection(sshTarget string, opts ConnectionOptions) Connection {
	execFn := ssh.ExecCmd
	if opts.WithMockExec != nil {
		execFn = opts.WithMockExec
	}
	return Connection{
		SSHTarget: ssh.Host(sshTarget),
		exec:      execFn,
		opts:      opts,
	}
}

func (c *Connection) Run(command string) (string, error) {
	if c.opts.WithLoginShell {
		command = ssh.ShellCommand(command)
	}

	sshArgs := []string{}
	if c.opts.Multiplex && runtime.GOOS != "windows" {
		sshArgs = append(sshArgs, "-o", "ControlMaster=auto", "-o", "ControlPersist=10s", "-o", "ControlPath=~/.ssh/topo-cm-%r@%h:%p")
	}

	cmd := c.exec(c.SSHTarget, command, c.opts.WithStdin, sshArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		combined := stdoutBuf.String() + stderrBuf.String()
		return combined, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", string(c.SSHTarget), err, stderrBuf.String())
	}
	return stdoutBuf.String(), nil
}

func (c *Connection) DryRun(command string, output io.Writer) error {
	if c.opts.WithLoginShell {
		command = ssh.ShellCommand(command)
	}

	cmd := c.exec(c.SSHTarget, command, c.opts.WithStdin)
	_, err := fmt.Fprintln(output, commandpkg.String(cmd))
	return err
}

func (c *Connection) BinaryExists(bin string) error {
	if err := ssh.ValidateBinaryName(bin); err != nil {
		return err
	}

	if _, err := c.Run(fmt.Sprintf("command -v %s", bin)); err != nil {
		return fmt.Errorf("%q executable file not found in $PATH", bin)
	}
	return nil
}

func (c *Connection) ProbeAuthentication() error {
	if !c.opts.AcceptNewHostKeys {
		err := c.runSSHAuthenticationProbe(knownHostProbeArgs)
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
	if err := c.runSSHAuthenticationProbe(slices.Concat(publicArgs, extraArgs)); err == nil {
		return false, nil
	} else if !errors.Is(err, ErrAuthenticationFailure) {
		return false, err
	}

	// Public key was rejected. Check if the target accepts password auth.
	passwordArgs := slices.Clone(passwordProbeArgs)
	if err := c.runSSHAuthenticationProbe(slices.Concat(passwordArgs, extraArgs)); err == nil {
		return false, nil
	} else if errors.Is(err, ErrAuthenticationFailure) {
		return true, nil
	} else {
		return false, err
	}
}

// All SSH authentication probes run the command "true" to check if the authentication method works.
// All sshArgs should be hardcoded SSH options, not user-provided arguments.
func (c *Connection) runSSHAuthenticationProbe(sshArgs []string) error {
	cmd := c.exec(c.SSHTarget, "true", nil, sshArgs...)
	stdoutBytes, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	output := strings.ToLower(string(stdoutBytes))
	if strings.Contains(output, "host key verification failed") {
		return ErrHostKeyVerification
	}
	if strings.Contains(output, "permission denied") || strings.Contains(output, "authentication failed") || strings.Contains(output, "password") {
		return ErrAuthenticationFailure
	}
	return fmt.Errorf("ssh probe failed: %w", err)
}
