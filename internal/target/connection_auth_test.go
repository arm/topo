package target_test

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sshTestCall struct {
	target  ssh.Destination
	command string
	args    []string
}

type mockResponse struct {
	stdout string
	err    error
}

func newMockExec(responses map[string]mockResponse, calls *[]sshTestCall) func(ssh.Destination, string, []byte, ...string) *exec.Cmd {
	return func(target ssh.Destination, command string, _ []byte, sshArgs ...string) *exec.Cmd {
		*calls = append(*calls, sshTestCall{
			target:  target,
			command: command,
			args:    sshArgs,
		})

		mode := "default"
		for _, arg := range sshArgs {
			switch arg {
			case "PreferredAuthentications=publickey":
				mode = "public"
			case "PreferredAuthentications=password":
				mode = "password"
			case "PasswordAuthentication=no":
				mode = "knownhost"
			}
		}

		resp, ok := responses[mode]
		if !ok {
			return testutil.CmdWithOutput(fmt.Sprintf("unexpected ssh mode: %s", mode), 1)
		}
		if resp.err != nil {
			return testutil.CmdWithOutput(resp.stdout, 1)
		}
		return testutil.CmdWithOutput(resp.stdout, 0)
	}
}

func TestProbeAuthentication(t *testing.T) {
	errSSH := errors.New("ssh failed")

	t.Run("does not require password when public key succeeds", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"public": {stdout: "", err: nil},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: true, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.NoError(t, err)

		require.Len(t, calls, 1)
		assert.Contains(t, calls[0].args, "PreferredAuthentications=publickey")
		assert.Contains(t, calls[0].args, "StrictHostKeyChecking=accept-new")
	})

	t.Run("returns host key verification error for public key probe", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"public": {stdout: "Host key verification failed", err: errSSH},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: true, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.ErrorIs(t, err, target.ErrHostKeyVerification)
		require.Len(t, calls, 1)
	})

	t.Run("returns host key verification error for password probe", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"public":   {stdout: "Permission denied", err: errSSH},
			"password": {stdout: "Host key verification failed", err: errSSH},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: true, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.ErrorIs(t, err, target.ErrHostKeyVerification)
		require.Len(t, calls, 2)
		assert.Contains(t, calls[1].args, "PreferredAuthentications=password")
	})

	t.Run("returns password-only auth error when auth fails", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"public":   {stdout: "Permission denied", err: errSSH},
			"password": {stdout: "Authentication failed", err: errSSH},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: true, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.ErrorIs(t, err, target.ErrPasswordAuthentication)
	})

	t.Run("does not require password when password probe succeeds", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"public":   {stdout: "Permission denied", err: errSSH},
			"password": {stdout: "ok", err: nil},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: true, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.NoError(t, err)
		require.Len(t, calls, 2)
		assert.Contains(t, calls[1].args, "PreferredAuthentications=password")
	})

	t.Run("returns error on non-auth failure for password probe", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"public":   {stdout: "Permission denied", err: errSSH},
			"password": {stdout: "Some other error", err: errSSH},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: true, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ssh probe failed")
	})

	t.Run("ensures known host when not accepting new host keys", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"knownhost": {stdout: "Permission denied", err: errSSH},
			"public":    {stdout: "", err: nil},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: false, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.NoError(t, err)

		require.Len(t, calls, 2)
		assert.Contains(t, calls[0].args, "PasswordAuthentication=no")
	})

	t.Run("returns host key verification error when known host fails", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"knownhost": {stdout: "HOST KEY VERIFICATION FAILED", err: errSSH},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: false, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.ErrorIs(t, err, target.ErrHostKeyVerification)
		require.Len(t, calls, 1)
	})

	t.Run("returns error when known host fails with other error", func(t *testing.T) {
		var calls []sshTestCall
		mockExec := newMockExec(map[string]mockResponse{
			"knownhost": {stdout: "dial tcp: lookup host: no such host", err: errSSH},
		}, &calls)
		opts := target.ConnectionOptions{AcceptNewHostKeys: false, WithMockExec: mockExec}
		conn := target.NewConnection(ssh.NewDestination("user@host"), opts)
		err := conn.ProbeAuthentication()
		require.Error(t, err)
		require.Len(t, calls, 1)
	})
}
