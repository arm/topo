package env_test

import (
	"fmt"
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

		assert.EqualError(t, err, fmt.Sprintf("env file %q does not exist", filepath.Join(root, "missing")))
		assert.Nil(t, paths)
	})

	t.Run("rejects invalid paths even when missing files are allowed", func(t *testing.T) {
		root := t.TempDir()

		paths, err := env.ResolveFiles(root, []string{"invalid\x00.env"}, true)

		require.ErrorContains(t, err, "failed to check env file")
		assert.Nil(t, paths)
	})
}
