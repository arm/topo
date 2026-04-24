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
		{name: "auth failure", stderr: "user@host: Permission denied (publickey).", want: ErrAuthFailed},
		{name: "too many authentication failures", stderr: "Received disconnect from 192.0.2.10 port 22:2: Too many authentication failures", want: ErrAuthFailed},
		{name: "connection refused", stderr: "ssh: connect to host example.com port 22: Connection refused", want: ErrConnectionFailed},
		{name: "hostname not resolved", stderr: "ssh: Could not resolve hostname not-valid: Temporary failure in name resolution", want: ErrConnectionFailed},
		{name: "operation timed out (macOS)", stderr: "ssh: connect to host example.com port 22: Operation timed out", want: ErrConnectionTimeout},
		{name: "connection timed out (Linux)", stderr: "ssh: connect to host example.com port 22: Connection timed out", want: ErrConnectionTimeout},
		{name: "generic host key verification failure", stderr: "Host key verification failed.", want: ErrHostKeyUnknown},
		{
			name: "unknown host key",
			stderr: `No ED25519 host key is known for 10.2.4.68 and you have requested strict checking.
Host key verification failed.`,
			want: ErrHostKeyUnknown,
		},
		{
			name: "host key has changed",
			stderr: `Host key for 10.2.4.68 has changed and you have requested strict checking.
Host key verification failed.`,
			want: ErrHostKeyChanged,
		},
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
