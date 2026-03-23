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
	"time"

	"github.com/arm/topo/internal/command"
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

type ExecSSH func(target ssh.Destination, cmdStr string, stdin []byte, sshArgs ...string) *exec.Cmd

type Connection struct {
	SSHTarget ssh.Destination
	exec      ExecSSH
	opts      ConnectionOptions
}

type ConnectionOptions struct {
	AcceptNewHostKeys bool
	WithLoginShell    bool
	WithStdin         []byte
	Multiplex         bool
	WithMockExec      ExecSSH
	ConnectTimeout    time.Duration
}

var ErrPasswordAuthentication = errors.New("key-based SSH authentication is not setup")

func NewConnection(dest ssh.Destination, opts ConnectionOptions) Connection {
	execFn := ssh.ExecCmd
	if opts.WithMockExec != nil {
		execFn = opts.WithMockExec
	}
	opts.ConnectTimeout = ssh.NewConfig(dest).ConnectTimeout(opts.ConnectTimeout)
	return Connection{
		SSHTarget: dest,
		exec:      execFn,
		opts:      opts,
	}
}

func (c *Connection) Run(cmdStr string) (string, error) {
	if c.opts.WithLoginShell {
		cmdStr = ssh.ShellCommand(cmdStr)
	}

	sshArgs := c.connectTimeoutArgs()
	if c.opts.Multiplex && runtime.GOOS != "windows" {
		sshArgs = append(sshArgs, "-o", "ControlMaster=auto", "-o", "ControlPersist=10s", "-o", "ControlPath=~/.ssh/topo-cm-%r@%h:%p")
	}

	cmd := c.exec(c.SSHTarget, cmdStr, c.opts.WithStdin, sshArgs...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		stderr := stderrBuf.String()
		if classified := ssh.ClassifyStderr(stderr); classified != nil {
			err = classified
		}
		return stdoutBuf.String() + stderr, fmt.Errorf("ssh command to %s failed: %w | stderr: %s", c.SSHTarget, err, stderr)
	}
	return stdoutBuf.String(), nil
}

func (c *Connection) DryRun(cmdStr string, output io.Writer) error {
	if c.opts.WithLoginShell {
		cmdStr = ssh.ShellCommand(cmdStr)
	}

	cmd := c.exec(c.SSHTarget, cmdStr, c.opts.WithStdin)
	_, err := fmt.Fprintln(output, command.String(cmd))
	return err
}

func (c *Connection) BinaryExists(bin string) error {
	if err := command.ValidateBinaryName(bin); err != nil {
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

type ConnectionTimeoutError struct {
	Timeout time.Duration
}

func (e ConnectionTimeoutError) Error() string {
	if e.Timeout > 0 {
		return fmt.Sprintf("ssh connection timed out after %s", e.Timeout)
	}
	return "ssh connection timed out"
}

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

func (c *Connection) connectTimeoutArgs() []string {
	if c.opts.ConnectTimeout <= 0 {
		return nil
	}
	return []string{"-o", fmt.Sprintf("ConnectTimeout=%d", int(c.opts.ConnectTimeout.Seconds()))}
}

// All SSH authentication probes run the command "true" to check if the authentication method works.
// All sshArgs should be hardcoded SSH options, not user-provided arguments.
func (c *Connection) runSSHAuthenticationProbe(sshArgs []string) error {
	cmd := c.exec(c.SSHTarget, "true", nil, slices.Concat(c.connectTimeoutArgs(), sshArgs)...)
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
	if strings.Contains(output, "timed out") || strings.Contains(output, "connection timeout") || strings.Contains(output, "did not properly respond after a period of time") {
		return ConnectionTimeoutError{Timeout: c.opts.ConnectTimeout}
	}
	return fmt.Errorf("ssh probe failed: %w", err)
}
