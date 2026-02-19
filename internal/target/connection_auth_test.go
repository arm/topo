package target_test

import (
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newConnectionWithOpts(opts target.ConnectionOptions) target.Connection {
	mockExec := func(_ ssh.Host, _ string) (string, error) {
		return "", nil
	}
	return target.NewConnection("user@host", mockExec, opts)
}

func TestProbeAuthentication(t *testing.T) {
	testutil.RequireOS(t, "linux")

	t.Run("does not require password when public key succeeds", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_PUBLIC_EXIT": "0",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: true})
			err := conn.ProbeAuthentication()
			require.NoError(t, err)

			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 1)
			assert.Contains(t, lines[0], "PreferredAuthentications=publickey")
			assert.Contains(t, lines[0], "StrictHostKeyChecking=accept-new")
		})
	})

	t.Run("returns host key verification error for public key probe", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_PUBLIC_STDERR": "Host key verification failed",
			"SSH_TEST_PUBLIC_EXIT":   "1",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: true})
			err := conn.ProbeAuthentication()
			require.ErrorIs(t, err, target.ErrHostKeyVerification)
			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 1)
		})
	})

	t.Run("returns host key verification error for password probe", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_PUBLIC_STDERR":   "Permission denied",
			"SSH_TEST_PUBLIC_EXIT":     "1",
			"SSH_TEST_PASSWORD_STDERR": "Host key verification failed",
			"SSH_TEST_PASSWORD_EXIT":   "1",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: true})
			err := conn.ProbeAuthentication()
			require.ErrorIs(t, err, target.ErrHostKeyVerification)
			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 2)
			assert.Contains(t, lines[1], "PreferredAuthentications=password")
		})
	})

	t.Run("returns password-only auth error when auth fails", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_PUBLIC_STDERR":   "Permission denied",
			"SSH_TEST_PUBLIC_EXIT":     "1",
			"SSH_TEST_PASSWORD_STDERR": "Authentication failed",
			"SSH_TEST_PASSWORD_EXIT":   "1",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: true})
			err := conn.ProbeAuthentication()
			require.ErrorIs(t, err, target.ErrPasswordAuthentication)
		})
	})

	t.Run("does not require password when password probe succeeds", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_PUBLIC_STDERR":   "Permission denied",
			"SSH_TEST_PUBLIC_EXIT":     "1",
			"SSH_TEST_PASSWORD_STDOUT": "ok",
			"SSH_TEST_PASSWORD_EXIT":   "0",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: true})
			err := conn.ProbeAuthentication()
			require.NoError(t, err)
			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 2)
			assert.Contains(t, lines[1], "PreferredAuthentications=password")
		})
	})

	t.Run("returns error on non-auth failure for password probe", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_PUBLIC_STDERR":   "Permission denied",
			"SSH_TEST_PUBLIC_EXIT":     "1",
			"SSH_TEST_PASSWORD_STDERR": "Some other error",
			"SSH_TEST_PASSWORD_EXIT":   "1",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: true})
			err := conn.ProbeAuthentication()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "ssh probe failed")
		})
	})

	t.Run("ensures known host when not accepting new host keys", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_KNOWNHOST_STDERR": "Permission denied",
			"SSH_TEST_KNOWNHOST_EXIT":   "1",
			"SSH_TEST_PUBLIC_EXIT":      "0",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: false})
			err := conn.ProbeAuthentication()
			require.NoError(t, err)

			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 2)
			assert.Contains(t, lines[0], "PasswordAuthentication=no")
		})
	})

	t.Run("returns host key verification error when known host fails", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_KNOWNHOST_STDERR": "HOST KEY VERIFICATION FAILED",
			"SSH_TEST_KNOWNHOST_EXIT":   "1",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: false})
			err := conn.ProbeAuthentication()
			require.ErrorIs(t, err, target.ErrHostKeyVerification)
			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 1)
		})
	})

	t.Run("returns error when known host fails with other error", func(t *testing.T) {
		testutil.WithFakeSSH(t, map[string]string{
			"SSH_TEST_KNOWNHOST_STDERR": "dial tcp: lookup host: no such host",
			"SSH_TEST_KNOWNHOST_EXIT":   "1",
		}, func(argsFile string) {
			conn := newConnectionWithOpts(target.ConnectionOptions{AuthProbeEnabled: true, AcceptNewHostKeys: false})
			err := conn.ProbeAuthentication()
			require.Error(t, err)
			lines := testutil.ReadArgsLines(t, argsFile)
			require.Len(t, lines, 1)
		})
	})
}
