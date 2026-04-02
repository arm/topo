package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyStderr(t *testing.T) {
	t.Run("returns ErrAuthFailed for publickey message", func(t *testing.T) {
		assert.ErrorIs(t, ClassifyStderr("Permission denied (publickey)"), ErrAuthFailed)
	})

	t.Run("returns ErrAuthFailed for authentication message", func(t *testing.T) {
		assert.ErrorIs(t, ClassifyStderr("Authentication failed"), ErrAuthFailed)
	})

	t.Run("returns ErrConnectionFailed for connection refused", func(t *testing.T) {
		assert.ErrorIs(t, ClassifyStderr("ssh: connect to host foo port 22: Connection refused"), ErrConnectionFailed)
	})

	t.Run("ErrAuthFailed satisfies ErrSSH", func(t *testing.T) {
		assert.ErrorIs(t, ErrAuthFailed, ErrSSH)
	})

	t.Run("ErrConnectionFailed satisfies ErrSSH", func(t *testing.T) {
		assert.ErrorIs(t, ErrConnectionFailed, ErrSSH)
	})

	t.Run("returns ErrConnectionTimeout for timed out message", func(t *testing.T) {
		assert.ErrorIs(t, ClassifyStderr("ssh: connect to host foo port 22: Operation timed out"), ErrConnectionTimeout)
	})

	t.Run("returns ErrConnectionTimeout for connection timeout message", func(t *testing.T) {
		assert.ErrorIs(t, ClassifyStderr("Connection timeout"), ErrConnectionTimeout)
	})

	t.Run("returns ErrConnectionTimeout for windows timeout message", func(t *testing.T) {
		assert.ErrorIs(t, ClassifyStderr("did not properly respond after a period of time"), ErrConnectionTimeout)
	})

	t.Run("ErrConnectionTimeout satisfies ErrSSH", func(t *testing.T) {
		assert.ErrorIs(t, ErrConnectionTimeout, ErrSSH)
	})

	t.Run("returns nil for unrecognised output", func(t *testing.T) {
		assert.Nil(t, ClassifyStderr("some unexpected error"))
	})

	t.Run("returns nil for empty stderr", func(t *testing.T) {
		assert.Nil(t, ClassifyStderr(""))
	})
}
