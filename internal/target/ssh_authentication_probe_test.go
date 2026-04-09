package target_test

import (
	"context"
	"errors"
	"testing"

	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type probeResult struct {
	output string
	err    error
}

type stubRunner struct {
	calls   [][]string
	results []probeResult
}

func (s *stubRunner) RunWithArgs(_ context.Context, _ string, sshArgs ...string) (string, error) {
	i := len(s.calls)
	s.calls = append(s.calls, sshArgs)
	return s.results[i].output, s.results[i].err
}

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
		method            target.ProbeMethod
		acceptNewHostKeys bool
		want              []string
	}{
		{
			name:   "public key probe",
			method: target.PublicKey,
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=publickey",
			},
		},
		{
			name:              "public key probe with accept new host keys",
			method:            target.PublicKey,
			acceptNewHostKeys: true,
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "StrictHostKeyChecking=accept-new",
			},
		},
		{
			name:   "password probe",
			method: target.Password,
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=password",
				"-o", "NumberOfPasswordPrompts=0",
			},
		},
		{
			name:              "password probe with accept new host keys",
			method:            target.Password,
			acceptNewHostKeys: true,
			want: []string{
				"-o", "BatchMode=yes",
				"-o", "PreferredAuthentications=password",
				"-o", "NumberOfPasswordPrompts=0",
				"-o", "StrictHostKeyChecking=accept-new",
			},
		},
		{
			name:   "known host probe",
			method: target.KnownHost,
			want: []string{
				"-o", "StrictHostKeyChecking=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "PasswordAuthentication=no",
				"-o", "NumberOfPasswordPrompts=0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := target.BuildProbeArgs(tt.method, tt.acceptNewHostKeys)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSSHAuthenticationProbe(t *testing.T) {
	errSSH := errors.New("ssh failed")
	authDenied := probeResult{output: "Permission denied", err: errSSH}
	success := probeResult{}
	hostKeyNew := probeResult{output: "Host key verification failed", err: errSSH}
	hostKeyChanged := probeResult{output: "Host key for 10.2.4.68 has changed.\nHost key verification failed.", err: errSSH}
	unknownFailure := probeResult{output: "connection reset", err: errSSH}

	t.Run("succeeds when public key authenticates", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{success}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.NoError(t, err)
		assert.Len(t, r.calls, 1)
	})

	t.Run("falls back to password when public key fails auth", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{authDenied, success}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.NoError(t, err)
		assert.Len(t, r.calls, 2)
	})

	t.Run("returns ErrPasswordAuthentication when both auth methods fail", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{authDenied, authDenied}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, target.ErrPasswordAuthentication)
	})

	t.Run("stops at public key when error is not auth failure", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{hostKeyNew}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, target.ErrHostKeyNew)
		assert.Len(t, r.calls, 1)
	})

	t.Run("returns password probe error when it is not auth failure", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{authDenied, unknownFailure}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, errSSH)
	})

	t.Run("returns host key error from password probe", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{authDenied, hostKeyChanged}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, target.ErrHostKeyChanged)
	})

	t.Run("skips known host check when accepting new host keys", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{success}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{AcceptNewHostKeys: true})

		err := probe.Probe(context.Background())

		require.NoError(t, err)
		assert.Len(t, r.calls, 1)
	})

	t.Run("runs known host check first when not accepting new host keys", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{authDenied, success}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{})

		err := probe.Probe(context.Background())

		require.NoError(t, err)
		assert.Len(t, r.calls, 2)
		assert.Contains(t, r.calls[0], "PasswordAuthentication=no")
	})

	t.Run("stops at known host check when host key is unknown", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{hostKeyNew}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, target.ErrHostKeyNew)
		assert.Len(t, r.calls, 1)
	})

	t.Run("stops at known host check when host key changed", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{hostKeyChanged}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, target.ErrHostKeyChanged)
		assert.Len(t, r.calls, 1)
	})

	t.Run("returns non-auth error from known host check", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{unknownFailure}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{})

		err := probe.Probe(context.Background())

		require.ErrorIs(t, err, errSSH)
		assert.Len(t, r.calls, 1)
	})

	t.Run("continues past known host check when auth is denied", func(t *testing.T) {
		r := &stubRunner{results: []probeResult{authDenied, success}}
		probe := target.NewSSHAuthenticationProbe(r, target.SSHAuthenticationProbeOptions{})

		err := probe.Probe(context.Background())

		require.NoError(t, err)
		assert.Len(t, r.calls, 2)
	})
}
