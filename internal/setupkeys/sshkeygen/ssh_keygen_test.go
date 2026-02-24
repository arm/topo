package sshkeygen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestSSHKeyGenDryRun(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "keys", "id_ed25519_test")
	op := sshkeygen.NewSSHKeyGen("Generate SSH key pair", "user@example.com", "ed25519", keyPath, sshkeygen.SSHKeyGenOptions{})

	var buf bytes.Buffer
	require.NoError(t, op.DryRun(&buf))
	require.Contains(t, buf.String(), "ssh-keygen -t ed25519 -f "+keyPath+" -C user@example.com")
}

func TestSSHKeyGenRun(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "keys", "id_ed25519_test")

	var got struct {
		keyType string
		keyPath string
		target  string
		called  bool
	}
	opts := sshkeygen.SSHKeyGenOptions{
		WithMockKeyGen: func(keyType, keyPath, targetHost string) *exec.Cmd {
			got = struct {
				keyType string
				keyPath string
				target  string
				called  bool
			}{keyType: keyType, keyPath: keyPath, target: targetHost, called: true}
			return testutil.CmdWithOutput("ssh-keygen invoked", 0)
		},
	}
	op := sshkeygen.NewSSHKeyGen("Generate SSH key pair", "user@example.com", "ed25519", keyPath, opts)

	var buf bytes.Buffer
	require.NoError(t, op.Run(&buf))
	require.True(t, got.called)
	require.Equal(t, "ed25519", got.keyType)
	require.Equal(t, keyPath, got.keyPath)
	require.Equal(t, "user@example.com", got.target)
	require.Contains(t, buf.String(), "ssh-keygen invoked")

	_, err := os.Stat(filepath.Dir(keyPath))
	require.NoError(t, err, "expected key directory to be created")
}
