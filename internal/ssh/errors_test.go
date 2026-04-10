package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyStderr(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		want   error
	}{
		{name: "publickey message", stderr: "Permission denied (publickey)", want: ErrAuthFailed},
		{name: "authentication message", stderr: "Authentication failed", want: ErrAuthFailed},
		{name: "permission denied", stderr: "Permission denied", want: ErrAuthFailed},
		{name: "password prompt", stderr: "password:", want: ErrAuthFailed},
		{name: "connection refused", stderr: "ssh: connect to host foo port 22: Connection refused", want: ErrConnectionFailed},
		{name: "timed out", stderr: "ssh: connect to host foo port 22: Operation timed out", want: ErrConnectionTimeout},
		{name: "connection timeout", stderr: "Connection timeout", want: ErrConnectionTimeout},
		{name: "windows timeout", stderr: "did not properly respond after a period of time", want: ErrConnectionTimeout},
		{name: "generic host key verification failure", stderr: "Host key verification failed.", want: ErrHostKeyUnknown},
		{name: "unknown host key", stderr: "No ED25519 host key is known for 10.2.4.68 and you have requested strict checking.\nHost key verification failed.", want: ErrHostKeyUnknown},
		{name: "host key has changed", stderr: "Host key for 10.2.4.68 has changed and you have requested strict checking.\nHost key verification failed.", want: ErrHostKeyChanged},
		{name: "case insensitive", stderr: "HOST KEY VERIFICATION FAILED", want: ErrHostKeyUnknown},
		{name: "unrecognised output", stderr: "some unexpected error", want: nil},
		{name: "empty stderr", stderr: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyStderr(tt.stderr)

			if tt.want == nil {
				assert.NoError(t, got)
			} else {
				assert.ErrorIs(t, got, tt.want)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{ErrAuthFailed, ErrConnectionFailed, ErrConnectionTimeout, ErrHostKeyUnknown, ErrHostKeyChanged}
	for _, err := range sentinels {
		t.Run(err.Error(), func(t *testing.T) {
			assert.ErrorIs(t, err, ErrSSH)
		})
	}
}
