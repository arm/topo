package setupkeys_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultPrivateKeyPath(t *testing.T) {
	t.Run("returns home-based ed25519 key path for target slug", func(t *testing.T) {
		tmp := t.TempDir()
		target := "user@some1thing.com"
		targetSlug := ssh.NewDestination(target).Slugify()

		got, err := setupkeys.GetDefaultPrivateKeyPath(tmp, targetSlug)

		require.NoError(t, err)
		require.Equal(t, filepath.Join(tmp, "id_ed25519_topo_user_some1thing.com"), got)
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
