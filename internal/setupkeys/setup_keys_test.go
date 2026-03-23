package setupkeys_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/setupkeys/pubkeytransfer"
	"github.com/arm/topo/internal/setupkeys/sshkeygen"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewKeySetup(t *testing.T) {
	t.Run("NewKeySetup returns SSHKeyGen then PubKeyTransfer for supported key types", func(t *testing.T) {
		t.Run("ed25519 with empty private key path", func(t *testing.T) {
			got, err := setupkeys.NewKeySetup(ssh.NewDestination("user@some1thing.com"), "", setupkeys.KeyTypeED25519)

			require.NoError(t, err)
			require.Len(t, got, 2)
			require.IsType(t, &sshkeygen.SSHKeyGen{}, got[0])
			require.IsType(t, &pubkeytransfer.PubKeyTransfer{}, got[1])
		})

		t.Run("ed25519 with custom private key path", func(t *testing.T) {
			got, err := setupkeys.NewKeySetup(
				ssh.NewDestination("user@some1thing.com"),
				filepath.Join(t.TempDir(), "id_ed25519_custom"),
				setupkeys.KeyTypeED25519,
			)

			require.NoError(t, err)
			require.Len(t, got, 2)
			require.IsType(t, &sshkeygen.SSHKeyGen{}, got[0])
			require.IsType(t, &pubkeytransfer.PubKeyTransfer{}, got[1])
		})

		t.Run("rsa with custom private key path", func(t *testing.T) {
			got, err := setupkeys.NewKeySetup(
				ssh.NewDestination("user@some2thing.com"),
				filepath.Join(t.TempDir(), "id_rsa_custom"),
				setupkeys.KeyTypeRSA,
			)

			require.NoError(t, err)
			require.Len(t, got, 2)
			require.IsType(t, &sshkeygen.SSHKeyGen{}, got[0])
			require.IsType(t, &pubkeytransfer.PubKeyTransfer{}, got[1])
		})
	})
}

func TestGetDefaultPrivateKeyPath(t *testing.T) {
	t.Run("returns home-based ed25519 key path for target slug", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		target := "user@some1thing.com"
		targetSlug := ssh.NewDestination(target).Slugify()

		got, err := setupkeys.GetDefaultPrivateKeyPath(targetSlug)

		require.NoError(t, err)
		require.Equal(t, filepath.Join(tmp, ".ssh", "id_ed25519_topo_user_some1thing.com"), got)
	})
}

func TestParseKeyType(t *testing.T) {
	t.Run("parses ed25519", func(t *testing.T) {
		got, err := setupkeys.ParseKeyType("ed25519")

		require.NoError(t, err)
		require.Equal(t, setupkeys.KeyTypeED25519, got)
	})

	t.Run("parses rsa", func(t *testing.T) {
		got, err := setupkeys.ParseKeyType("rsa")

		require.NoError(t, err)
		require.Equal(t, setupkeys.KeyTypeRSA, got)
	})

	t.Run("returns error for unsupported key type ecdsa", func(t *testing.T) {
		_, err := setupkeys.ParseKeyType("ecdsa")

		require.EqualError(t, err, `unsupported key type "ecdsa", supported types: ed25519, rsa`)
	})
}
