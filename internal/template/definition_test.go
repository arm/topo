package template_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromContent(t *testing.T) {
	t.Run("parses multiple service definitions", func(t *testing.T) {
		composeFileContents := `
services:
  app1:
    image: nginx:alpine
  app2:
    image: redis:alpine
`
		tpl, err := template.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Services

		require.NoError(t, err)
		want := []template.Service{
			{
				Name: "app1",
				Data: map[string]any{
					"image": "nginx:alpine",
				},
			},
			{
				Name: "app2",
				Data: map[string]any{
					"image": "redis:alpine",
				},
			},
		}
		assert.ElementsMatch(t, want, got)
	})

	t.Run("parses x-topo metadata", func(t *testing.T) {
		composeFileContents := `
  x-topo:
    name: "test-service"
    description: "Test service"
    features:
      - "SME"
      - "NEON"
`
		tpl, err := template.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Metadata

		require.NoError(t, err)
		want := template.Metadata{
			Name:        "test-service",
			Description: "Test service",
			Features:    []string{"SME", "NEON"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses args from x-topo metadata", func(t *testing.T) {
		composeFileContents := `
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
		tpl, err := template.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Metadata.Args

		require.NoError(t, err)
		want := []template.Arg{
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
		}
		assert.Equal(t, want, got)
	})

	t.Run("errors when compose.yaml missing", func(t *testing.T) {
		dir := t.TempDir()

		_, err := template.FromDir(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), template.ComposeFilename)
	})
}

func TestFromDir(t *testing.T) {
	t.Run("finds a compose file in directory and parses into template", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app1:
    image: nginx:alpine

x-topo:
  args:
    GREETING:
      description: "The greeting message to display"
      required: true
      example: "Hello, World"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, template.ComposeFilename), composeFileContents)

		got, err := template.FromDir(dir)

		require.NoError(t, err)
		want, _ := template.FromContent(strings.NewReader(composeFileContents))
		assert.Equal(t, want, got)
	})
}
