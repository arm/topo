package project_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
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
		tpl, err := project.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Services

		require.NoError(t, err)
		want := []project.Service{
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
		tpl, err := project.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Metadata

		require.NoError(t, err)
		want := project.Metadata{
			Name:        "test-service",
			Description: "Test service",
			Features:    []string{"SME", "NEON"},
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses parameters from x-topo metadata", func(t *testing.T) {
		composeFileContents := `
  x-topo:
    parameters:
      GREETING:
        description: "The greeting message to display"
        required: true
        example: "Hello, World"
      PORT:
        description: "Port number"
        required: false
  `
		tpl, err := project.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Metadata.Parameters

		require.NoError(t, err)
		want := []project.Parameter{
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

	t.Run("parses deployment_success_message from x-topo metadata", func(t *testing.T) {
		composeFileContents := `
x-topo:
  name: "test-service"
  deployment_success_message: "Deployment complete!"
`
		tpl, err := project.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Metadata

		require.NoError(t, err)
		want := project.Metadata{
			Name:                     "test-service",
			DeploymentSuccessMessage: "Deployment complete!",
		}
		assert.Equal(t, want, got)
	})

	t.Run("leaves DeploymentSuccessMessage empty when deployment_success_message absent from x-topo", func(t *testing.T) {
		composeFileContents := `
x-topo:
  name: "test-service"
`
		tpl, err := project.FromContent(strings.NewReader(composeFileContents))
		got := tpl.Metadata

		require.NoError(t, err)
		assert.Empty(t, got.DeploymentSuccessMessage)
	})

	t.Run("errors when compose.yaml missing", func(t *testing.T) {
		dir := t.TempDir()

		_, err := project.FromDir(dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), project.ComposeFilename)
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
  parameters:
    GREETING:
      description: "The greeting message to display"
      required: true
      example: "Hello, World"
`
		testutil.RequireWriteFile(t, filepath.Join(dir, project.ComposeFilename), composeFileContents)

		got, err := project.FromDir(dir)

		require.NoError(t, err)
		want, _ := project.FromContent(strings.NewReader(composeFileContents))
		assert.Equal(t, want, got)
	})
}
