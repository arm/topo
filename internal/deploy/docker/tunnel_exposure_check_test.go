package docker_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckTunnelExposure(t *testing.T) {
	t.Run("fails when the SSH hostname cannot be resolved", func(t *testing.T) {
		err := docker.CheckTunnelExposure(context.Background(), io.Discard, ssh.Destination{}, "12345")

		assert.ErrorContains(t, err, "cannot conclusively rule out network access to registry port 12345")
		assert.ErrorContains(t, err, `could not resolve SSH configuration for "ssh://"`)
		assert.ErrorContains(t, err, "use `--skip-remote-port-check` if you understand the security risk")
	})

	t.Run("succeeds when the remote port refuses the connection", func(t *testing.T) {
		port := testutil.RequireAvailableTCPPort(t, "0.0.0.0")
		var output strings.Builder

		err := docker.CheckTunnelExposure(context.Background(), &output, ssh.NewDestination("0.0.0.0"), port)

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

		err = docker.CheckTunnelExposure(context.Background(), io.Discard, ssh.NewDestination("localhost."), port)
		listenerCloseErr := listener.Close()

		assert.EqualError(t, err, fmt.Sprintf("the remote SSH server is exposing forwarded registry port %s beyond remote loopback; configure the SSH server to bind remote forwards to loopback only, or use `--skip-remote-port-check` if you understand that the registry may be reachable without SSH authentication", port))
		assert.NoError(t, listenerCloseErr)
		assert.NoError(t, <-connectionClosed)
	})
}
