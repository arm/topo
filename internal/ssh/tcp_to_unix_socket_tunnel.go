package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"time"

	"github.com/arm/topo/internal/command"
)

const tcpTunnelOpenTimeout = 10 * time.Second

// TCPToUnixSocketTunnel forwards a loopback TCP endpoint to a Unix socket on an
// SSH destination. A TCP endpoint is used locally so the transport works on
// Unix and Windows hosts.
type TCPToUnixSocketTunnel struct {
	command   *exec.Cmd
	socketURL string
	closed    bool
}

func OpenTCPToUnixSocketTunnel(ctx context.Context, output io.Writer, dest Destination, remoteSocketPath string) (*TCPToUnixSocketTunnel, error) {
	address, err := availableLoopbackAddress()
	if err != nil {
		return nil, err
	}

	args := []string{
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-L", fmt.Sprintf("%s:%s", address, remoteSocketPath),
		dest.String(),
	}
	cmd := exec.CommandContext(ctx, "ssh", args...) // #nosec G204 -- arguments are not interpreted by a shell
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Start(); err != nil {
		return nil, command.FormatError(cmd.Args, err)
	}

	tunnel := &TCPToUnixSocketTunnel{
		command:   cmd,
		socketURL: "tcp://" + address,
	}
	if err := waitForTCPListener(ctx, address); err != nil {
		closeErr := tunnel.Close()
		return nil, errors.Join(fmt.Errorf("failed to wait for SSH tunnel: %w", err), closeErr)
	}
	return tunnel, nil
}

func (t *TCPToUnixSocketTunnel) SocketURL() string {
	return t.socketURL
}

func (t *TCPToUnixSocketTunnel) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	return killCommand(t.command)
}

func availableLoopbackAddress() (string, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to reserve a local loopback address for the SSH tunnel: %w", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", fmt.Errorf("failed to release local loopback address %s for the SSH tunnel: %w", address, err)
	}
	return address, nil
}

func waitForTCPListener(ctx context.Context, address string) error {
	deadline := time.NewTimer(tcpTunnelOpenTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for {
		connection, err := net.DialTimeout("tcp4", address, 100*time.Millisecond)
		if err == nil {
			return connection.Close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("timed out waiting for %s", address)
		case <-ticker.C:
		}
	}
}
