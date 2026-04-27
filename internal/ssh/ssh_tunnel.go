package ssh

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/operation"
)

const TunnelPIDPlaceholder = "<ssh tunnel pid>"

func isPortTaken(port string) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%s", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close() // nolint:errcheck
	return true
}

func ControlSocketPath(targetHost string) string {
	hash := sha256.Sum256([]byte(targetHost))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

func NewSSHTunnel(targetDest Destination, port string, useControlSockets bool) (operation.Operation, operation.Operation, operation.Operation) {
	start := NewSSHTunnelStart(targetDest, port, useControlSockets)
	securityCheck := NewCheckSSHTunnelSecurity(targetDest, port)

	var stop operation.Operation
	if useControlSockets {
		stop = NewSSHTunnelStop(targetDest)
	} else {
		stop = NewSSHTunnelProcessStop(start)
	}

	return start, securityCheck, stop
}

type SSHTunnelStart struct {
	TargetDest        Destination
	UseControlSockets bool
	Port              string
	Process           *os.Process
}

func NewSSHTunnelStart(targetDest Destination, port string, useControlSockets bool) *SSHTunnelStart {
	return &SSHTunnelStart{TargetDest: targetDest, Port: port, UseControlSockets: useControlSockets}
}

func (s *SSHTunnelStart) Description() string {
	return "Open registry SSH tunnel"
}

func (s *SSHTunnelStart) Command() *exec.Cmd {
	args := []string{"ssh", "-N", "-o", "ExitOnForwardFailure=yes"}
	if s.UseControlSockets {
		args = append(args,
			"-fMS", ControlSocketPath(s.TargetDest.String()),
		)
	}
	args = append(args,
		"-R", fmt.Sprintf("%s:127.0.0.1:%s", s.Port, s.Port),
		s.TargetDest.String(),
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
		if isPortTaken(s.Port) {
			return fmt.Errorf("port already in use: %s - specify a different port or stop the existing process", s.Port)
		}

		formattedError := command.FormatError(cmd.Args, err)
		return fmt.Errorf("failed to open ssh tunnel: %w", formattedError)
	}
	if cmd.Process != nil {
		s.Process = cmd.Process
	}
	_, _ = fmt.Fprintln(w, "Tunnel created")
	return nil
}

type CheckSSHTunnelSecurity struct {
	TargetDest Destination
	Port       string
}

func NewCheckSSHTunnelSecurity(targetDest Destination, port string) *CheckSSHTunnelSecurity {
	return &CheckSSHTunnelSecurity{TargetDest: targetDest, Port: port}
}

func (ct *CheckSSHTunnelSecurity) Description() string {
	return "Check SSH tunnel security"
}

func (ct *CheckSSHTunnelSecurity) Run(w io.Writer) error {
	if ct.TargetDest.IsLocalhost() {
		return nil
	}
	cmd := ct.command()
	cmd.Stdout = w
	cmd.Stderr = w

	err := cmd.Run()
	if err == nil {
		return fmt.Errorf("ssh tunnel to %s is not secure: able to access registry port without authentication", ct.TargetDest)
	}

	_, _ = fmt.Fprintf(w, "Port %s is not exposed on target to local network\n", ct.Port)
	return nil
}

func (ct *CheckSSHTunnelSecurity) command() *exec.Cmd {
	hostName := NewConfig(ct.TargetDest).HostName
	return exec.Command("curl", fmt.Sprintf("%s:%s", hostName, ct.Port), "--max-time", "1")
}

type SSHTunnelStop struct {
	TargetDest Destination
}

func NewSSHTunnelStop(targetDest Destination) *SSHTunnelStop {
	return &SSHTunnelStop{TargetDest: targetDest}
}

func (s *SSHTunnelStop) Description() string {
	return "Close registry SSH tunnel"
}

func (s *SSHTunnelStop) Command() *exec.Cmd {
	args := []string{"ssh"}
	args = append(args,
		"-S", ControlSocketPath(s.TargetDest.String()),
		"-O", "exit",
		s.TargetDest.String(),
	)
	// #nosec -- arguments are validated
	return exec.Command(args[0], args[1:]...)
}

func (s *SSHTunnelStop) Run(w io.Writer) error {
	if _, err := os.Stat(ControlSocketPath(s.TargetDest.String())); os.IsNotExist(err) {
		return nil
	}
	cmd := s.Command()
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		formattedError := command.FormatError(cmd.Args, err)
		return fmt.Errorf("failed to close SSH tunnel: %w", formattedError)
	}
	return nil
}

type SSHTunnelProcessStop struct {
	Start *SSHTunnelStart
}

func NewSSHTunnelProcessStop(start *SSHTunnelStart) *SSHTunnelProcessStop {
	return &SSHTunnelProcessStop{Start: start}
}

func (s *SSHTunnelProcessStop) Description() string {
	return "Close registry SSH tunnel"
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
		pid := s.Start.Process.Pid
		formattedError := command.FormatError(cmd.Args, err)
		return fmt.Errorf("failed to stop ssh tunnel process (pid: %d): %w", pid, formattedError)
	}
	s.Start.Process = nil
	return nil
}
