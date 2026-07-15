package ssh

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arm/topo/internal/command"
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

	securityCheck := NewCheckRemoteForwardNotExposed(targetDest, port)

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
		"-R", fmt.Sprintf("127.0.0.1:%s:127.0.0.1:%s", s.Port, s.Port),
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
		formattedError := command.FormatError(cmd.Args, err)
		return fmt.Errorf("failed to open ssh tunnel: %w - ensure port %s is free or specify a different one with `--registry-port`", formattedError, s.Port)
	}
	if cmd.Process != nil {
		s.Process = cmd.Process
	}
	_, _ = fmt.Fprintln(w, "Tunnel created")
	return nil
}

type CheckRemoteForwardNotExposed struct {
	TargetDest Destination
	Port       string
}

// Checks whether the RemoteForward port exposes the registry to the target's
// network, rather than being limited to target loopback. This can happen when
// sshd permits non-loopback remote forwards, such as GatewayPorts.
func NewCheckRemoteForwardNotExposed(targetDest Destination, port string) *CheckRemoteForwardNotExposed {
	return &CheckRemoteForwardNotExposed{TargetDest: targetDest, Port: port}
}

func (ct *CheckRemoteForwardNotExposed) Description() string {
	return "Check tunnel port is not exposed on remote network"
}

func (ct *CheckRemoteForwardNotExposed) Run(w io.Writer) error {
	if ct.TargetDest.IsLocalhost() {
		return nil
	}

	host, err := ResolveHostName(ct.TargetDest)
	if err != nil {
		return remotePortCheckErrorWithSuggestion(err, ct.Port)
	}
	err, inconclusive := checkRemotePortNotListening(host, ct.Port)
	if inconclusive {
		return remotePortCheckErrorWithSuggestion(err, ct.Port)
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(w, "Port %s is bound to remote loopback only\n", ct.Port)
	return nil
}

func checkRemotePortNotListening(host, port string) (error, bool) {
	address := net.JoinHostPort(host, port)
	var remoteIP strings.Builder
	cmd := exec.Command("curl", "http://"+address, "--max-time", "5", "--noproxy", "*", "--output", os.DevNull, "--silent", "--write-out", "%{remote_ip}")
	cmd.Stdout = &remoteIP
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if err == nil || remoteIP.Len() > 0 {
		return fmt.Errorf("remote sshd might be exposing the forwarded port %s on its network (likely GatewayPorts=yes); the local registry may be reachable without SSH auth", port), false
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 7 {
		return nil, false
	}
	if errors.As(err, &exitError) && exitError.ExitCode() == 28 {
		return fmt.Errorf("timed out while checking whether remote port %s is exposed: %w", address, err), true
	}
	if errors.As(err, &exitError) && exitError.ExitCode() == 6 {
		return fmt.Errorf("could not resolve remote host %q while checking tunnel exposure: %w", host, err), true
	}

	return fmt.Errorf("could not verify whether remote port %s is exposed: %w", address, err), true
}

func remotePortCheckErrorWithSuggestion(err error, port string) error {
	return fmt.Errorf("cannot conclusively rule out network access to registry port %s because the exposure check did not complete: %w; retry after resolving the connectivity issue, or use `--skip-remote-port-check` if you understand the security risk", port, err)
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
