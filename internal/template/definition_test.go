package template_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDefinition(t *testing.T) {
	t.Run("parses file to ComposeFile", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app:
    image: nginx:alpine
    ports:
      - "8000:80"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, template.ComposeFilename), composeFileContents)

		got, err := template.ParseDefinition(dir)

		require.NoError(t, err)
		want := template.ComposeFile{
			Services: map[string]any{
				"app": map[string]any{
					"image": "nginx:alpine",
					"ports": []any{"8000:80"},
				},
			},
			XTopo: template.Metadata{},
		}
		assert.Equal(t, want, got)
	})
}

func TestParseComposeFileToTemplates(t *testing.T) {
	t.Run("parses multiple service definitions", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app1:
    image: nginx:alpine
  app2:
    image: redis:alpine
`
		testutil.RequireWriteFile(t, filepath.Join(dir, template.ComposeFilename), composeFileContents)

		got, err := template.ParseComposeFileToTemplates(dir)

		require.NoError(t, err)
		sort.Slice(got, func(i, j int) bool {
			return got[i].ServiceName < got[j].ServiceName
		})

		want := []template.Template{
			{
				Service: map[string]any{
					"image": "nginx:alpine",
				},
				ServiceName: "app1",
			},
			{
				Service: map[string]any{
					"image": "redis:alpine",
				},
				ServiceName: "app2",
			},
		}
		sort.Slice(want, func(i, j int) bool {
			return want[i].ServiceName < want[j].ServiceName
		})

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
		testutil.RequireWriteFile(t, filepath.Join(dir, template.ComposeFilename), composeFileContents)

		got, err := template.ParseComposeFileToTemplates(dir)

		require.NoError(t, err)
		want := []template.Template{
			{
				Service: map[string]any{
					"image": "nginx:alpine",
				},
				ServiceName: "app",
				Metadata: template.Metadata{
					Name:        "test-service",
					Description: "Test service",
					Features:    []string{"SME", "NEON"},
				},
			},
		}
		assert.Equal(t, want, got)
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
		testutil.RequireWriteFile(t, filepath.Join(dir, template.ComposeFilename), composeFileContents)

		got, err := template.ParseComposeFileToTemplates(dir)

		require.NoError(t, err)
		want := []template.Template{
			{
				Metadata: template.Metadata{
					Args: []template.Arg{
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
				},
				Service: map[string]any{
					"image": "nginx:alpine",
				},
				ServiceName: "app",
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("errors when compose.yaml missing", func(t *testing.T) {
		dir := t.TempDir()

		_, err := template.ParseComposeFileToTemplates(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), template.ComposeFilename)
	})
}
