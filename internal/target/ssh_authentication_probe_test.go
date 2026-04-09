package target_test

import (
	"errors"
	"testing"

	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifySSHOutput(t *testing.T) {
	errSSH := errors.New("ssh failed")

	tests := []struct {
		name   string
		output string
		err    error
		want   error
	}{
		{
			name: "returns nil when no error",
			err:  nil,
			want: nil,
		},
		{
			name:   "returns ErrHostKeyNew for generic host key verification failure",
			output: "Host key verification failed.",
			err:    errSSH,
			want:   target.ErrHostKeyNew,
		},
		{
			name:   "returns ErrHostKeyNew for unknown host key",
			output: "No ED25519 host key is known for 10.2.4.68 and you have requested strict checking.\nHost key verification failed.",
			err:    errSSH,
			want:   target.ErrHostKeyNew,
		},
		{
			name:   "returns ErrHostKeyChanged when host key has changed",
			output: "Host key for 10.2.4.68 has changed and you have requested strict checking.\nHost key verification failed.",
			err:    errSSH,
			want:   target.ErrHostKeyChanged,
		},
		{
			name:   "returns ErrAuthenticationFailure for permission denied",
			output: "Permission denied",
			err:    errSSH,
			want:   target.ErrAuthenticationFailure,
		},
		{
			name:   "returns ErrAuthenticationFailure for authentication failed",
			output: "Authentication failed",
			err:    errSSH,
			want:   target.ErrAuthenticationFailure,
		},
		{
			name:   "returns ErrAuthenticationFailure for password prompt",
			output: "password:",
			err:    errSSH,
			want:   target.ErrAuthenticationFailure,
		},
		{
			name:   "is case insensitive",
			output: "HOST KEY VERIFICATION FAILED",
			err:    errSSH,
			want:   target.ErrHostKeyNew,
		},
		{
			name:   "returns original error for unrecognized output",
			output: "dial tcp: lookup host: no such host",
			err:    errSSH,
			want:   errSSH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := target.ClassifySSHOutput(tt.output, tt.err)

			if tt.want == nil {
				require.NoError(t, got)
			} else {
				require.ErrorIs(t, got, tt.want)
			}
		})
	}
}

func TestBuildProbeArgs(t *testing.T) {
	tests := []struct {
		name              string
		acceptNewHostKeys bool
		want              []string
	}{
		{
			name: "strict host key checking by default",
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-o", "NumberOfPasswordPrompts=0",
				"-o", "StrictHostKeyChecking=yes",
			},
		},
		{
			name:              "accept new host keys",
			acceptNewHostKeys: true,
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-o", "NumberOfPasswordPrompts=0",
				"-o", "StrictHostKeyChecking=accept-new",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := target.BuildProbeArgs(tt.acceptNewHostKeys)

			assert.Equal(t, tt.want, got)
		})
	}
}


