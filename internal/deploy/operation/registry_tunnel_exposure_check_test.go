package operation_test

import (
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/operation"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryTunnelExposureCheck(t *testing.T) {
	t.Run("returns the expected description", func(t *testing.T) {
		check := operation.NewRegistryTunnelExposureCheck(ssh.NewDestination("user@remote"), "12345")

		got := check.Description()

		assert.Equal(t, "Check registry tunnel is not exposed on remote network", got)
	})

	t.Run("skips the check for localhost", func(t *testing.T) {
		check := operation.NewRegistryTunnelExposureCheck(ssh.PlainLocalhost, "invalid")
		var output strings.Builder

		err := check.Run(&output)

		assert.NoError(t, err)
		assert.Empty(t, output.String())
	})

	t.Run("fails when the SSH hostname cannot be resolved", func(t *testing.T) {
		check := operation.NewRegistryTunnelExposureCheck(ssh.Destination{}, "12345")

		err := check.Run(io.Discard)

		assert.ErrorContains(t, err, "cannot conclusively rule out network access to registry port 12345")
		assert.ErrorContains(t, err, `could not resolve SSH configuration for "ssh://"`)
		assert.ErrorContains(t, err, "use `--skip-remote-port-check` if you understand the security risk")
	})

	t.Run("succeeds when the remote port refuses the connection", func(t *testing.T) {
		port := reserveFreePort(t, "0.0.0.0")
		check := operation.NewRegistryTunnelExposureCheck(ssh.NewDestination("0.0.0.0"), port)
		var output strings.Builder

		err := check.Run(&output)

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("Registry port %s is bound to remote loopback only\n", port), output.String())
	})

	t.Run("fails when the remote port accepts a connection", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })
		connectionClosed := make(chan error, 1)
		go func() {
			connection, acceptErr := listener.Accept()
			if acceptErr != nil {
				connectionClosed <- acceptErr
				return
			}
			connectionClosed <- connection.Close()
		}()
		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)
		check := operation.NewRegistryTunnelExposureCheck(ssh.NewDestination("localhost."), port)

		err = check.Run(io.Discard)
		listenerCloseErr := listener.Close()

		assert.EqualError(t, err, fmt.Sprintf("the remote SSH server is exposing forwarded registry port %s beyond remote loopback; configure the SSH server to bind remote forwards to loopback only, or use `--skip-remote-port-check` if you understand that the registry may be reachable without SSH authentication", port))
		assert.NoError(t, listenerCloseErr)
		assert.NoError(t, <-connectionClosed)
	})
}

func reserveFreePort(t *testing.T, host string) string {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	return port
}
