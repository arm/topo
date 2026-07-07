package compose_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireFile(t *testing.T) {
	t.Run("returns path when compose file exists", func(t *testing.T) {
		composeFile := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, composeFile, "")

		got, err := compose.RequireFile(composeFile)

		require.NoError(t, err)
		assert.Equal(t, composeFile, got)
	})

	t.Run("returns error when compose file does not exist", func(t *testing.T) {
		composeFile := filepath.Join(t.TempDir(), "compose.yaml")

		got, err := compose.RequireFile(composeFile)

		require.Error(t, err)
		assert.Empty(t, got)
		assert.ErrorContains(t, err, "compose file not found")
		assert.ErrorContains(t, err, composeFile)
	})

	t.Run("returns error when compose file path is a directory", func(t *testing.T) {
		composeFile := filepath.Join(t.TempDir(), "compose.yaml")
		require.NoError(t, os.Mkdir(composeFile, 0o755))

		got, err := compose.RequireFile(composeFile)

		require.Error(t, err)
		assert.Empty(t, got)
		assert.ErrorContains(t, err, "compose file path is a directory")
		assert.ErrorContains(t, err, composeFile)
	})
}

func TestFindDefaultFile(t *testing.T) {
	t.Run("returns compose.yaml when it exists", func(t *testing.T) {
		t.Chdir(t.TempDir())
		testutil.RequireWriteFile(t, "compose.yaml", "")

		got, err := compose.FindDefaultFile()

		require.NoError(t, err)
		assert.Equal(t, "compose.yaml", got)
	})

	t.Run("falls back to compose.yml", func(t *testing.T) {
		t.Chdir(t.TempDir())
		testutil.RequireWriteFile(t, "compose.yml", "")

		got, err := compose.FindDefaultFile()

		require.NoError(t, err)
		assert.Equal(t, "compose.yml", got)
	})

	t.Run("warns when multiple default compose files exist", func(t *testing.T) {
		t.Chdir(t.TempDir())
		testutil.RequireWriteFile(t, "compose.yaml", "")
		testutil.RequireWriteFile(t, "compose.yml", "")
		var logOutput bytes.Buffer
		logger.SetOptions(logger.Options{Output: &logOutput, Format: term.Plain})
		t.Cleanup(func() {
			logger.SetOptions(logger.Options{})
		})

		got, err := compose.FindDefaultFile()

		require.NoError(t, err)
		assert.Equal(t, "compose.yaml", got)
		assert.Contains(t, logOutput.String(), "found multiple compose files: compose.yaml, compose.yml; using compose.yaml")
	})

	t.Run("returns error when no default compose file exists", func(t *testing.T) {
		t.Chdir(t.TempDir())

		got, err := compose.FindDefaultFile()

		require.Error(t, err)
		assert.Empty(t, got)
		assert.EqualError(t, err, "compose file not found in current working directory: looking for compose.yaml or compose.yml")
	})
}
