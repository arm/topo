package project_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
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
		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Services

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
		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Metadata

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
		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Metadata.Parameters

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

	t.Run("parses legacy args from x-topo metadata when parameters are absent", func(t *testing.T) {
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
		var logOutput bytes.Buffer
		logger.SetOptions(logger.Options{Output: &logOutput, Format: term.Plain})
		t.Cleanup(func() {
			logger.SetOptions(logger.Options{})
		})

		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Metadata.Parameters

		require.NoError(t, err)
		assert.Contains(t, logOutput.String(), "x-topo.args is deprecated; use x-topo.parameters instead")
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

	t.Run("parses parameters aliased to legacy args", func(t *testing.T) {
		composeFileContents := `
  x-topo:
    args: &args
      GREETING:
        description: "The greeting message to display"
        required: true
        example: "Hello, World"
      PORT:
        description: "Port number"
        required: false
    parameters: *args
  `
		var logOutput bytes.Buffer
		logger.SetOptions(logger.Options{Output: &logOutput, Format: term.Plain})
		t.Cleanup(func() {
			logger.SetOptions(logger.Options{})
		})

		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Metadata.Parameters

		require.NoError(t, err)
		assert.Empty(t, logOutput.String())
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
		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Metadata

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
		p, err := project.FromContent(strings.NewReader(composeFileContents))
		got := p.Metadata

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
	t.Run("finds a compose file in directory and parses into project", func(t *testing.T) {
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
