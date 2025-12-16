package ssh

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

const RegistryPort = 12737 // Not 5000 to try to avoid conflicts with the user.

func ControlSocketPath(targetHost string) string {
	hash := sha256.Sum256([]byte(targetHost))
	hostHash := fmt.Sprintf("%x", hash[:8]) // Hash to avoid filepath limits
	return filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%s", hostHash))
}

func splitHostPort(raw string) (host, port string) {
	userPart := ""
	hostPart := raw
	if at := strings.LastIndex(raw, "@"); at != -1 {
		userPart = raw[:at+1]
		hostPart = raw[at+1:]
	}

	host, port, err := net.SplitHostPort(hostPart)
	if err == nil {
		if strings.HasPrefix(hostPart, "[") {
			host = "[" + host + "]"
		}
		return userPart + host, port
	}
	return raw, ""
}

type SSHTunnelStart struct {
	TargetHost Host
}

func (s *SSHTunnelStart) Description() string {
	return "Open registry SSH tunnel"
}

func NewSSHTunnelStart(targetHost Host) *SSHTunnelStart {
	return &SSHTunnelStart{TargetHost: targetHost}
}

func (s *SSHTunnelStart) Command() *exec.Cmd {
	host, port := splitHostPort(string(s.TargetHost))
	args := []string{"ssh", "-MNf", "-o", "ExitOnForwardFailure=yes"}
	if port != "" {
		args = append(args, "-p", port)
	}
	args = append(args,
		"-S", ControlSocketPath(string(s.TargetHost)),
		"-L", fmt.Sprintf("%d:localhost:%d", RegistryPort, RegistryPort),
		host,
	)
	return exec.Command(args[0], args[1:]...)
}

func (s *SSHTunnelStart) Run(w io.Writer) error {
	cmd := s.Command()
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open SSH tunnel to %s: %w", s.TargetHost, err)
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
	host, port := splitHostPort(string(s.TargetHost))
	args := []string{"ssh"}
	if port != "" {
		args = append(args, "-p", port)
	}
	args = append(args,
		"-S", ControlSocketPath(string(s.TargetHost)),
		"-O", "exit",
		host,
	)
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

func CloseTunnel(targetHost Host, w io.Writer) {
	if err := NewSSHTunnelStop(targetHost).Run(w); err != nil {
		_, _ = fmt.Fprintf(w, "Warning: failed to close SSH tunnel to %s: %v\n", targetHost, err)
	}
}

// Sets up signal handling to ensure the SSH tunnel is properly closed
// on interrupt or termination. Returns a cancel function to stop signal handling and a
// cleanup function to close the tunnel.
func SetupTunnelCleanup(targetHost Host, w io.Writer) (cancel func(), cleanup func()) {
	var closeOnce sync.Once
	closeTunnelOnce := func() {
		closeOnce.Do(func() {
			CloseTunnel(targetHost, w)
		})
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig, ok := <-sigChan
		if !ok || sig == nil {
			return
		}
		closeTunnelOnce()
		os.Exit(1)
	}()
	cancel = func() {
		signal.Stop(sigChan)
		close(sigChan)
	}
	return cancel, closeTunnelOnce
}
