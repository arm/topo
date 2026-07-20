package netprobe_test

import (
	"context"
	"net"
	"testing"

	"github.com/arm/topo/internal/netprobe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRemotePortListening(t *testing.T) {
	t.Run("returns false when the remote port refuses the connection", func(t *testing.T) {
		port := reserveFreePort(t, "0.0.0.0")

		got, err := netprobe.IsRemotePortListening(context.Background(), "0.0.0.0", port)

		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("returns true when the remote port accepts a connection", func(t *testing.T) {
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

		got, err := netprobe.IsRemotePortListening(context.Background(), "localhost.", port)
		listenerCloseErr := listener.Close()

		assert.NoError(t, err)
		assert.True(t, got)
		assert.NoError(t, listenerCloseErr)
		assert.NoError(t, <-connectionClosed)
	})

	t.Run("returns an error when the remote host cannot be resolved", func(t *testing.T) {
		got, err := netprobe.IsRemotePortListening(context.Background(), "nonexistent.invalid", "12345")

		assert.ErrorContains(t, err, `could not resolve remote host "nonexistent.invalid"`)
		assert.False(t, got)
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
