packageetestutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func RequireAvailableTCPPort(t testing.TB, host string) string {
	t.Helper()
	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
	return port
}
