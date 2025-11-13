package service

import (
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDefinition(t *testing.T) {
	t.Run("parses service definition", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app:
    image: nginx:alpine
    ports:
      - "8000:80"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, ComposeServiceFilename), composeFileContents)

		got, err := ParseDefinition(dir)

		require.NoError(t, err)
		want := TemplateManifest{
			Service: map[string]any{
				"image": "nginx:alpine",
				"ports": []any{"8000:80"},
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses x-topo metadata", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "test-service"
  description: "Test service"
  features:
    - "SME"
    - "NEON"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, ComposeServiceFilename), composeFileContents)

		got, err := ParseDefinition(dir)

		require.NoError(t, err)
		want := TopoMetadata{
			Name:        "test-service",
			Description: "Test service",
			Features:    []string{"SME", "NEON"},
		}
		assert.Equal(t, want, got.Metadata)
	})

	t.Run("parses args from x-topo metadata", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app:
    image: nginx:alpine

x-topo:
  args:
    GREETING:
      description: "The greeting message to display"
      required: true
      example: "Hello, World"
    PORT:
      description: "Port number"
      required: false
`
		testutil.RequireWriteFile(t, filepath.Join(dir, ComposeServiceFilename), composeFileContents)

		got, err := ParseDefinition(dir)

		require.NoError(t, err)
		want := TopoMetadata{
			Args: map[string]ArgMetadata{
				"GREETING": {
					Description: "The greeting message to display",
					Required:    true,
					Example:     "Hello, World",
				},
				"PORT": {
					Description: "Port number",
					Required:    false,
				},
			},
		}
		assert.Equal(t, want, got.Metadata)
	})

	t.Run("errors when compose.service.yaml missing", func(t *testing.T) {
		dir := t.TempDir()

		_, err := ParseDefinition(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), ComposeServiceFilename)
	})

	t.Run("errors when no services defined", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
x-topo:
  name: "test-service"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, ComposeServiceFilename), composeFileContents)

		_, err := ParseDefinition(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no services defined")
	})

	t.Run("errors when multiple services defined", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app1:
    image: nginx:alpine
  app2:
    image: redis:alpine
`
		testutil.RequireWriteFile(t, filepath.Join(dir, ComposeServiceFilename), composeFileContents)

		_, err := ParseDefinition(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected exactly one service")
	})
}
