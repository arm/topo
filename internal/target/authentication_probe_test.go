package target_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSSHRunner struct {
	mock.Mock
}

func (m *mockSSHRunner) RunWithArgs(command string, sshArgs ...string) (string, error) {
	ret := m.Called(command, sshArgs)
	return ret.String(0), ret.Error(1)
}

var (
	publicKeyMode = mock.MatchedBy(func(args []string) bool {
		return slices.Contains(args, "PreferredAuthentications=publickey") &&
			!slices.Contains(args, "PasswordAuthentication=no")
	})
	passwordMode = mock.MatchedBy(func(args []string) bool {
		return slices.Contains(args, "PreferredAuthentications=password")
	})
	knownHostMode = mock.MatchedBy(func(args []string) bool {
		return slices.Contains(args, "PasswordAuthentication=no")
	})
)

func TestAuthenticationProbe(t *testing.T) {
	errSSH := errors.New("ssh failed")

	t.Run("does not require password when public key succeeds", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", publicKeyMode).Return("", nil)

		probe := target.NewAuthenticationProbe(runner, true)
		err := probe.Probe()

		require.NoError(t, err)
		runner.AssertExpectations(t)
		call := runner.Calls[0]
		assert.Contains(t, call.Arguments[1], "StrictHostKeyChecking=accept-new")
	})

	t.Run("returns host key verification error for public key probe", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", publicKeyMode).Return("Host key verification failed", errSSH)

		probe := target.NewAuthenticationProbe(runner, true)
		err := probe.Probe()

		require.ErrorIs(t, err, target.ErrHostKeyVerification)
		runner.AssertNumberOfCalls(t, "RunWithArgs", 1)
	})

	t.Run("returns host key verification error for password probe", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", publicKeyMode).Return("Permission denied", errSSH)
		runner.On("RunWithArgs", "true", passwordMode).Return("Host key verification failed", errSSH)

		probe := target.NewAuthenticationProbe(runner, true)
		err := probe.Probe()

		require.ErrorIs(t, err, target.ErrHostKeyVerification)
		runner.AssertNumberOfCalls(t, "RunWithArgs", 2)
	})

	t.Run("returns password-only auth error when auth fails", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", publicKeyMode).Return("Permission denied", errSSH)
		runner.On("RunWithArgs", "true", passwordMode).Return("Authentication failed", errSSH)

		probe := target.NewAuthenticationProbe(runner, true)
		err := probe.Probe()

		require.ErrorIs(t, err, target.ErrPasswordAuthentication)
	})

	t.Run("does not require password when password probe succeeds", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", publicKeyMode).Return("Permission denied", errSSH)
		runner.On("RunWithArgs", "true", passwordMode).Return("", nil)

		probe := target.NewAuthenticationProbe(runner, true)
		err := probe.Probe()

		require.NoError(t, err)
		runner.AssertExpectations(t)
	})

	t.Run("returns error on non-auth failure for password probe", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", publicKeyMode).Return("Permission denied", errSSH)
		runner.On("RunWithArgs", "true", passwordMode).Return("Some other error", errSSH)

		probe := target.NewAuthenticationProbe(runner, true)
		err := probe.Probe()

		require.ErrorIs(t, err, errSSH)
	})

	t.Run("ensures known host when not accepting new host keys", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", knownHostMode).Return("Permission denied", errSSH)
		runner.On("RunWithArgs", "true", publicKeyMode).Return("", nil)

		probe := target.NewAuthenticationProbe(runner, false)
		err := probe.Probe()

		require.NoError(t, err)
		runner.AssertExpectations(t)
		assert.Contains(t, runner.Calls[0].Arguments[1], "PasswordAuthentication=no")
	})

	t.Run("returns host key verification error when known host fails", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", knownHostMode).Return("HOST KEY VERIFICATION FAILED", errSSH)

		probe := target.NewAuthenticationProbe(runner, false)
		err := probe.Probe()

		require.ErrorIs(t, err, target.ErrHostKeyVerification)
		runner.AssertNumberOfCalls(t, "RunWithArgs", 1)
	})

	t.Run("returns error when known host fails with other error", func(t *testing.T) {
		runner := &mockSSHRunner{}
		runner.On("RunWithArgs", "true", knownHostMode).Return("dial tcp: lookup host: no such host", errSSH)

		probe := target.NewAuthenticationProbe(runner, false)
		err := probe.Probe()

		require.ErrorIs(t, err, errSSH)
		runner.AssertNumberOfCalls(t, "RunWithArgs", 1)
	})
}
