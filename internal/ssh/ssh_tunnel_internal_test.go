package ssh

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
