package ssh

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arm/topo/internal/operation"
)

const TunnelPIDPlaceholder = "<ssh tunnel pid>"

func ControlSocketPath(targetHost string) string {
	hash := sha256.Sum256([]byte(targetHost))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

func formatSSHHost(raw string, user string, host string) string {
	if host == "" {
		return raw
	}
	hostPart := host
	if strings.Contains(hostPart, ":") {
		hostPart = "[" + hostPart + "]"
	}
	if user == "" {
		return hostPart
	}
	return user + "@" + hostPart
}

func NewSSHTunnel(targetHost Host, port string, useControlSockets bool) (operation.Operation, operation.Operation) {
	start := NewSSHTunnelStart(targetHost, port, useControlSockets)
	var stop operation.Operation
	if useControlSockets {
		stop = NewSSHTunnelStop(targetHost)
	} else {
		stop = NewSSHTunnelProcessStop(start)
	}

	return start, stop
}

type SSHTunnelStart struct {
	TargetHost        Host
	UseControlSockets bool
	Port              string
	Process           *os.Process
}

func (s *SSHTunnelStart) Description() string {
	return "Open registry SSH tunnel"
}

func NewSSHTunnelStart(targetHost Host, port string, useControlSockets bool) *SSHTunnelStart {
	return &SSHTunnelStart{TargetHost: targetHost, Port: port, UseControlSockets: useControlSockets}
}

func (s *SSHTunnelStart) Command() *exec.Cmd {
	rawHost := string(s.TargetHost)
	user, host, port := SplitUserHostPort(rawHost)
	hostArg := formatSSHHost(rawHost, user, host)
	args := []string{"ssh", "-N", "-o", "ExitOnForwardFailure=yes"}
	if port != "" {
		args = append(args, "-p", port)
	}
	if s.UseControlSockets {
		args = append(args,
			"-fMS", ControlSocketPath(string(s.TargetHost)),
		)
	}
	args = append(args,
		"-R", fmt.Sprintf("%s:127.0.0.1:%s", s.Port, s.Port),
		hostArg,
	)
	// #nosec -- arguments are validated
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

type SSHTunnelStop struct {
	TargetHost Host
}

func (s *SSHTunnelStop) Description() string {
	return "Close registry SSH tunnel"
}

func NewSSHTunnelStop(targetHost Host) *SSHTunnelStop {
	return &SSHTunnelStop{TargetHost: targetHost}
}

func (s *SSHTunnelStop) Command() *exec.Cmd {
	rawHost := string(s.TargetHost)
	user, host, port := SplitUserHostPort(rawHost)
	hostArg := formatSSHHost(rawHost, user, host)
	args := []string{"ssh"}
	if port != "" {
		args = append(args, "-p", port)
	}
	args = append(args,
		"-S", ControlSocketPath(string(s.TargetHost)),
		"-O", "exit",
		hostArg,
	)
	// #nosec -- arguments are validated
	return exec.Command(args[0], args[1:]...)
}

func (s *SSHTunnelStop) Run(w io.Writer) error {
	if _, err := os.Stat(ControlSocketPath(string(s.TargetHost))); os.IsNotExist(err) {
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
