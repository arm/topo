package project_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromContent(t *testing.T) {
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
}
