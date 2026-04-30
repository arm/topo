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

	var probeHost string
	if !targetDest.IsLocalhost() {
		probeHost = NewConfig(targetDest).HostName
	}
	securityCheck := NewCheckRemoteForwardNotExposed(probeHost, port)

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

type CheckRemoteForwardNotExposed struct {
	Host string
	Port string
}

// Checks whether the RemoteForward port exposes the registry to the target's
// network, rather than being limited to target loopback. This can happen when
// sshd permits non-loopback remote forwards, such as GatewayPorts.
func NewCheckRemoteForwardNotExposed(host, port string) *CheckRemoteForwardNotExposed {
	return &CheckRemoteForwardNotExposed{Host: host, Port: port}
}

func (ct *CheckRemoteForwardNotExposed) Description() string {
	return "Check tunnel port is not exposed on remote network"
}

func (ct *CheckRemoteForwardNotExposed) Run(w io.Writer) error {
	if ct.Host == "" {
		return nil
	}
	if IsRemotePortDefinitelyNotListening(ct.Host, ct.Port) {
		_, _ = fmt.Fprintf(w, "Port %s is bound to remote loopback only\n", ct.Port)
		return nil
	}
	return fmt.Errorf("remote sshd might be exposing the forwarded port %s on its network (likely GatewayPorts=yes); the local registry may be reachable without SSH auth", ct.Port)
}

func IsRemotePortDefinitelyNotListening(host, port string) bool {
	// TODO: use curl.exe on windows
	cmd := exec.Command("curl", fmt.Sprintf("%s:%s", host, port), "--max-time", "1")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	_ = cmd.Run()

	// Only curl exit 7 ("Failed to connect to host") proves no TCP listener
	// answered. Anything else — connection succeeded, DNS failure, timeout,
	// curl missing — leaves us unable to certify the port is not not listening.
	if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 7 {
		return true
	}
	return false
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
