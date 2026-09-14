package env_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFiles(t *testing.T) {
	t.Run("resolves files relative to root", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, ".env.first"), "")
		testutil.RequireWriteFile(t, filepath.Join(root, ".env.second"), "")

		paths, err := env.ResolveFiles(root, []string{".env.second", ".env.first"}, false)

		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, ".env.second"), filepath.Join(root, ".env.first")}, paths)
	})

	t.Run("skips missing optional files", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, ".env"), "")

		paths, err := env.ResolveFiles(root, []string{"missing", ".env", "also-missing"}, true)

		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, ".env")}, paths)
	})

	t.Run("rejects missing required files without partial results", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, ".env"), "")

		paths, err := env.ResolveFiles(root, []string{".env", "missing"}, false)

		assert.EqualError(t, err, "env file \""+filepath.Join(root, "missing")+"\" does not exist")
		assert.Nil(t, paths)
	})

	t.Run("does not suppress other filesystem errors for optional files", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, "not-a-directory"), "")

		paths, err := env.ResolveFiles(root, []string{"not-a-directory/file.env"}, true)

		require.ErrorContains(t, err, "failed to check env file")
		assert.Nil(t, paths)
	})
}
