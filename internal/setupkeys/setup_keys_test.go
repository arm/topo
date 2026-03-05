package setupkeys_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewKeySetupReturnsExpectedSequence(t *testing.T) {
	tests := []struct {
		name        string
		keyType     string
		privKeyPath string
		target      string
	}{
		{
			name:        "ed25519 key type",
			keyType:     "ed25519",
			privKeyPath: filepath.Join(t.TempDir(), "id_ed25519_custom"),
			target:      "user@some1thing.com",
		},
		{
			name:        "rsa key type",
			keyType:     "rsa",
			privKeyPath: filepath.Join(t.TempDir(), "id_rsa_custom"),
			target:      "user@some2thing.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setupkeys.NewKeySetup(tt.target, tt.privKeyPath, tt.keyType)
			require.NoError(t, err)
			require.Len(t, got, 2)
			require.IsType(t, &sshkeygen.SSHKeyGen{}, got[0])
			require.IsType(t, &pubkeytransfer.PubKeyTransfer{}, got[1])
		})
	}
}

func TestGetDefaultPrivateKeyPath(t *testing.T) {
	tmp := t.TempDir()
	testutil.SetHomeDir(t, tmp)

	target := "user@some1thing.com"
	targetSlug := ssh.Host(target).Slugify()

	got, err := setupkeys.GetDefaultPrivateKeyPath(targetSlug)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(tmp, ".ssh", "id_ed25519_topo_user_some1thing.com"), got)
}

func TestSSHKeyGenOperationDryRunAndDescription(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "custom_keys", "id_ed25519_custom")
	op := sshkeygen.NewSSHKeyGen(
		"Generate SSH key pair for target",
		"user@some2thing.com",
		"ed25519",
		keyPath,
		sshkeygen.SSHKeyGenOptions{},
	)

	require.Equal(t, "Generate SSH key pair for target", op.Description())

	var buf bytes.Buffer
	require.NoError(t, op.DryRun(&buf))
	require.Contains(t, buf.String(), "ssh-keygen -t ed25519 -f "+keyPath+" -C user@some2thing.com")
}

func TestPubKeyTransferOperationDryRunAndDescription(t *testing.T) {
	privKeyPath := filepath.Join(t.TempDir(), "custom_keys", "id_ed25519_custom")
	op := pubkeytransfer.NewPubKeyTransfer(
		"Transfer public key to target and set it as an authorized key",
		"user@some3thing.com",
		privKeyPath,
		pubkeytransfer.PubKeyTransferOptions{},
	)

	require.Equal(t, "Transfer public key to target and set it as an authorized key", op.Description())

	var buf bytes.Buffer
	require.NoError(t, op.DryRun(&buf))
	require.Contains(t, buf.String(), "ssh -- user@some3thing.com")
	require.Contains(t, buf.String(), "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")
}

func TestNewKeySetupUnsupportedKeyType(t *testing.T) {
	_, err := setupkeys.NewKeySetup("user@example.com", "/tmp/id_invalid", "arrivederci")
	require.ErrorContains(t, err, "unsupported key type")
	require.ErrorContains(t, err, "arrivederci")
}
