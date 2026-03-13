package ssh

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/arm/topo/internal/operation"
)

const TunnelPIDPlaceholder = "<ssh tunnel pid>"

func NewSSHTunnel(targetHost SSHConnection, registryPort string, useControlSockets bool) (operation.Operation, operation.Operation, operation.Operation) {
	start := NewSSHTunnelStart(targetHost, registryPort, useControlSockets)
	securityCheck := NewCheckSSHTunnelSecurity(targetHost, registryPort)

	var stop operation.Operation
	if useControlSockets {
		stop = NewSSHTunnelStop(targetHost)
	} else {
		stop = NewSSHTunnelProcessStop(start)
	}

	return start, securityCheck, stop
}

type SSHTunnelStart struct {
	TargetHost        SSHConnection
	UseControlSockets bool
	RegistryPort      string
	Process           *os.Process
}

func (s *SSHTunnelStart) Description() string {
	return "Open registry SSH tunnel"
}

func NewSSHTunnelStart(targetHost SSHConnection, registryPort string, useControlSockets bool) *SSHTunnelStart {
	return &SSHTunnelStart{TargetHost: targetHost, RegistryPort: registryPort, UseControlSockets: useControlSockets}
}

func (s *SSHTunnelStart) Command() *exec.Cmd {
	args := s.TargetHost.FormatSSHConnectCommand(s.UseControlSockets, s.RegistryPort)
	return exec.Command(args[0], args[1:]...)
}

func (s *SSHTunnelStart) Run(w io.Writer) error {
	cmd := s.Command()
	cmd.Stdout = w
	cmd.Stderr = w
	run := cmd.Start
	if s.UseControlSockets {
		run = cmd.Run
	}
	if err := run(); err != nil {
		return fmt.Errorf("failed to open SSH tunnel to %s: %w", s.TargetHost, err)
	}
	if cmd.Process != nil {
		s.Process = cmd.Process
	}
	_, _ = fmt.Fprintln(w, "Tunnel created")
	return nil
}

func (s *SSHTunnelStart) DryRun(w io.Writer) error {
	_, _ = fmt.Fprintln(w, strings.Join(s.Command().Args, " "))
	return nil
}

type CheckSSHTunnelSecurity struct {
	TargetHost SSHConnection
	Port       string
}

func (ct *CheckSSHTunnelSecurity) Description() string {
	return "Check SSH tunnel security"
}

func NewCheckSSHTunnelSecurity(targetHost SSHConnection, port string) *CheckSSHTunnelSecurity {
	return &CheckSSHTunnelSecurity{TargetHost: targetHost, Port: port}
}

func (ct *CheckSSHTunnelSecurity) Command() *exec.Cmd {
	if !ct.TargetHost.IsLocalhost() {
		host := ct.TargetHost.GetHost()
		if host == "" {
			return nil
		}
		return exec.Command("curl", fmt.Sprintf("%s:%s", host, ct.Port), "--max-time", "1")
	}
	return nil
}

func (ct *CheckSSHTunnelSecurity) Run(w io.Writer) error {
	if ct.TargetHost.IsLocalhost() {
		return nil
	}
	cmd := ct.Command()
	if cmd == nil {
		panic(fmt.Sprintf("BUG: security check called for unresolvable host %q; caller must validate host before invoking", ct.TargetHost))
	}
	cmd.Stdout = w
	cmd.Stderr = w

	err := cmd.Run()
	if err == nil {
		return fmt.Errorf("SSH tunnel to %s is not secure: able to access registry port without authentication", ct.TargetHost)
	}

	return nil
}

func (ct *CheckSSHTunnelSecurity) DryRun(w io.Writer) error {
	if ct.TargetHost.IsLocalhost() {
		return nil
	}
	_, err := fmt.Fprintln(w, strings.Join(ct.Command().Args, " "))
	return err
}

type SSHTunnelStop struct {
	TargetHost SSHConnection
}

func (s *SSHTunnelStop) Description() string {
	return "Close registry SSH tunnel"
}

func NewSSHTunnelStop(targetHost SSHConnection) *SSHTunnelStop {
	return &SSHTunnelStop{TargetHost: targetHost}
}

func (s *SSHTunnelStop) Command() *exec.Cmd {
	args := s.TargetHost.FormatSSHExitCommand()
	return exec.Command(args[0], args[1:]...)
}

func (s *SSHTunnelStop) Run(w io.Writer) error {
	if _, err := os.Stat(s.TargetHost.ControlSocketPath()); os.IsNotExist(err) {
		return nil
	}
	cmd := s.Command()
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to close SSH tunnel to %s: %w", s.TargetHost, err)
	}
	return nil
}

func (s *SSHTunnelStop) DryRun(w io.Writer) error {
	_, _ = fmt.Fprintln(w, strings.Join(s.Command().Args, " "))
	return nil
}

type SSHTunnelProcessStop struct {
	Start *SSHTunnelStart
}

func (s *SSHTunnelProcessStop) Description() string {
	return "Close registry SSH tunnel"
}

func NewSSHTunnelProcessStop(start *SSHTunnelStart) *SSHTunnelProcessStop {
	return &SSHTunnelProcessStop{Start: start}
}

func (s *SSHTunnelProcessStop) Command() *exec.Cmd {
	pid := TunnelPIDPlaceholder
	if s.Start != nil && s.Start.Process != nil {
		pid = fmt.Sprintf("%d", s.Start.Process.Pid)
	}

	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/PID", pid, "/F")
	}
	return exec.Command("kill", "-9", pid)
}

func (s *SSHTunnelProcessStop) Run(w io.Writer) error {
	if s.Start == nil || s.Start.Process == nil {
		return nil
	}

	cmd := s.Command()
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop SSH tunnel process: %d", s.Start.Process.Pid)
	}
	s.Start.Process = nil
	return nil
}

func (s *SSHTunnelProcessStop) DryRun(w io.Writer) error {
	_, _ = fmt.Fprintln(w, strings.Join(s.Command().Args, " "))
	return nil
}

func resolveHost(raw string) string {
	if raw == "" || isExplicitHost(raw) {
		_, host, _ := SplitUserHostPort(raw)
		return host
	}

	config := NewConfig(raw)
	return config.host
}

func isExplicitHost(raw string) bool {
	if strings.HasPrefix(raw, "ssh://") {
		return true
	}
	if strings.Contains(raw, "@") || strings.Contains(raw, ":") {
		return true
	}
	if strings.HasPrefix(raw, "[") {
		return true
	}
	return false
}
