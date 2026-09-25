package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSources(t *testing.T) {
	t.Run("later assignment wins even when values are identical", func(t *testing.T) {
		dir := t.TempDir()
		first := filepath.Join(dir, ".env")
		second := filepath.Join(dir, ".env.topo")
		require.NoError(t, os.WriteFile(first, []byte("TOPO_SOURCE_TEST=same\nTOPO_FIRST_ONLY=one\n"), 0600))
		require.NoError(t, os.WriteFile(second, []byte("TOPO_SOURCE_TEST=same\n"), 0600))
		values, err := env.CurrentValues([]string{first, second})
		require.NoError(t, err)
		sources, err := env.Sources([]string{first, second}, values)
		require.NoError(t, err)
		assert.Equal(t, second, sources["TOPO_SOURCE_TEST"])
		assert.Equal(t, first, sources["TOPO_FIRST_ONLY"])
	})
	t.Run("empty shell assignment overrides file", func(t *testing.T) {
		t.Setenv("TOPO_SOURCE_TEST", "")
		path := filepath.Join(t.TempDir(), ".env")
		require.NoError(t, os.WriteFile(path, []byte("TOPO_SOURCE_TEST=file\n"), 0600))
		sources, err := env.Sources([]string{path}, map[string]string{"TOPO_SOURCE_TEST": ""})
		require.NoError(t, err)
		assert.Equal(t, "shell environment", sources["TOPO_SOURCE_TEST"])
	})
	t.Run("missing file returns an error", func(t *testing.T) {
		_, err := env.Sources([]string{filepath.Join(t.TempDir(), "missing")}, nil)
		assert.Error(t, err)
	})
}
