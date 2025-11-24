package service_test

import (
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/service"
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
		testutil.RequireWriteFile(t, filepath.Join(dir, service.ComposeServiceFilename), composeFileContents)

		got, err := service.ParseDefinition(dir)

		require.NoError(t, err)
		want := service.Template{
			Service: map[string]any{
				"image": "nginx:alpine",
				"ports": []any{"8000:80"},
			},
			ServiceName: "app",
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
		testutil.RequireWriteFile(t, filepath.Join(dir, service.ComposeServiceFilename), composeFileContents)

		got, err := service.ParseDefinition(dir)

		require.NoError(t, err)
		want := service.Metadata{
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
		testutil.RequireWriteFile(t, filepath.Join(dir, service.ComposeServiceFilename), composeFileContents)

		got, err := service.ParseDefinition(dir)

		require.NoError(t, err)
		want := service.Metadata{
			Args: []service.Arg{
				{
					Name:        "GREETING",
					Description: "The greeting message to display",
					Required:    true,
					Example:     "Hello, World",
				},
				{
					Name:        "PORT",
					Description: "Port number",
					Required:    false,
				},
			},
		}
		assert.Equal(t, want, got.Metadata)
	})

	t.Run("errors when compose.service.yaml missing", func(t *testing.T) {
		dir := t.TempDir()

		_, err := service.ParseDefinition(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), service.ComposeServiceFilename)
	})

	t.Run("errors when no services defined", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
x-topo:
  name: "test-service"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, service.ComposeServiceFilename), composeFileContents)

		_, err := service.ParseDefinition(dir)

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
		testutil.RequireWriteFile(t, filepath.Join(dir, service.ComposeServiceFilename), composeFileContents)

		_, err := service.ParseDefinition(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected exactly one service")
	})
}
