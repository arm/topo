package target

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
)

type ExecSSH func(target ssh.Destination, cmdStr string, stdin []byte, sshArgs ...string) *exec.Cmd

type Connection struct {
	SSHTarget ssh.Destination
	exec      ExecSSH
	opts      ConnectionOptions
}

type ConnectionOptions struct {
	WithLoginShell bool
	WithStdin      []byte
	Multiplex      bool
	WithMockExec   ExecSSH
	ConnectTimeout time.Duration
}

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

func (c *Connection) RunWithArgs(cmdStr string, sshArgs ...string) (string, error) {
	allArgs := append(c.connectTimeoutArgs(), sshArgs...)
	cmd := c.exec(c.SSHTarget, cmdStr, nil, allArgs...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	output := strings.ToLower(string(out))
	if strings.Contains(output, "timed out") || strings.Contains(output, "connection timeout") || strings.Contains(output, "did not properly respond after a period of time") {
		return string(out), ConnectionTimeoutError{Timeout: c.opts.ConnectTimeout}
	}
	return string(out), err
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

type ConnectionTimeoutError struct {
	Timeout time.Duration
}

func (e ConnectionTimeoutError) Error() string {
	if e.Timeout > 0 {
		return fmt.Sprintf("ssh connection timed out after %s", e.Timeout)
	}
	return "ssh connection timed out"
}

func (c *Connection) connectTimeoutArgs() []string {
	if c.opts.ConnectTimeout <= 0 {
		return nil
	}
	return []string{"-o", fmt.Sprintf("ConnectTimeout=%d", int(c.opts.ConnectTimeout.Seconds()))}
}
