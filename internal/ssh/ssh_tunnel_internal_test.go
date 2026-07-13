package ssh

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRemotePortNotListening(t *testing.T) {
	t.Run("succeeds when nothing answers on the remote port", func(t *testing.T) {
		port := reserveFreePort(t)

		err := checkRemotePortNotListening("127.0.0.1", port)

		assert.NoError(t, err)
	})

	t.Run("reports an exposed port when a TCP listener accepts the connection", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		t.Cleanup(func() { _ = listener.Close() })
		_, port, err := net.SplitHostPort(listener.Addr().String())
		require.NoError(t, err)

		err = checkRemotePortNotListening("127.0.0.1", port)

		assert.EqualError(t, err, fmt.Sprintf("remote sshd might be exposing the forwarded port %s on its network (likely GatewayPorts=yes); the local registry may be reachable without SSH auth", port))
	})

	t.Run("reports a resolution error when the remote host does not resolve", func(t *testing.T) {
		err := checkRemotePortNotListening("nonexistent.invalid", "12345")

		assert.ErrorContains(t, err, `could not resolve remote host "nonexistent.invalid" while checking tunnel exposure`)
	})
}

func reserveFreePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	return port
}

func TestClassifyRemotePortError(t *testing.T) {
	t.Run("returns a timeout-specific error for DNS timeouts", func(t *testing.T) {
		dialError := &net.DNSError{
			Err:       "operation timed out",
			Name:      "remote.example",
			IsTimeout: true,
		}

		err := classifyRemotePortError("remote.example", "remote.example:12345", dialError)

		assert.ErrorContains(t, err, "timed out while checking whether remote port remote.example:12345 is exposed")
		assert.ErrorIs(t, err, dialError)
	})

	t.Run("returns a fallback error for an unknown network failure", func(t *testing.T) {
		dialError := errors.New("unexpected network failure")

		err := classifyRemotePortError("remote.example", "remote.example:12345", dialError)

		assert.ErrorContains(t, err, "could not verify whether remote port remote.example:12345 is exposed")
		assert.ErrorIs(t, err, dialError)
	})
}
