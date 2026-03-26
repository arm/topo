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

func (s *SSHTunnelStart) Description() string {
	return "Open registry SSH tunnel"
}

func NewSSHTunnelStart(targetDest Destination, port string, useControlSockets bool) *SSHTunnelStart {
	return &SSHTunnelStart{TargetDest: targetDest, Port: port, UseControlSockets: useControlSockets}
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
		return fmt.Errorf("failed to open SSH tunnel to %s: %w", s.TargetDest, err)
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
	TargetDest Destination
	Port       string
}

func (ct *CheckSSHTunnelSecurity) Description() string {
	return "Check SSH tunnel security"
}

func NewCheckSSHTunnelSecurity(targetDest Destination, port string) *CheckSSHTunnelSecurity {
	return &CheckSSHTunnelSecurity{TargetDest: targetDest, Port: port}
}

func (ct *CheckSSHTunnelSecurity) Command() *exec.Cmd {
	if !ct.TargetDest.IsLocalhost() {
		hostName := NewConfig(ct.TargetDest).HostName
		return exec.Command("curl", fmt.Sprintf("%s:%s", hostName, ct.Port), "--max-time", "1")
	}
	return nil
}

func (ct *CheckSSHTunnelSecurity) Run(w io.Writer) error {
	if ct.TargetDest.IsLocalhost() {
		return nil
	}
	cmd := ct.Command()
	if cmd == nil {
		panic(fmt.Sprintf("BUG: security check called for unresolvable host %q; caller must validate host before invoking", ct.TargetDest))
	}

	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err == nil {
		_, _ = fmt.Fprint(w, buf.String())
		return fmt.Errorf("SSH tunnel to %s is not secure: able to access registry port without authentication", ct.TargetDest)
	}

	_, _ = fmt.Fprintf(w, "Port %s is not exposed on target to local network\n", ct.Port)
	return nil
}

func (ct *CheckSSHTunnelSecurity) DryRun(w io.Writer) error {
	if ct.TargetDest.IsLocalhost() {
		return nil
	}
	_, err := fmt.Fprintln(w, strings.Join(ct.Command().Args, " "))
	return err
}

type SSHTunnelStop struct {
	TargetDest Destination
}

func (s *SSHTunnelStop) Description() string {
	return "Close registry SSH tunnel"
}

func NewSSHTunnelStop(targetDest Destination) *SSHTunnelStop {
	return &SSHTunnelStop{TargetDest: targetDest}
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
		return fmt.Errorf("failed to close SSH tunnel to %s: %w", s.TargetDest, err)
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
