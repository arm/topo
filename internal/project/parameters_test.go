package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadParameterDefinitions(t *testing.T) {
	t.Run("loads parameter metadata", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-topo:
  parameters:
    PORT:
      description: HTTP port
      required: true
      example: "8080"
    EMPTY: {}
    UNSET: {}
`)

		got, err := project.LoadParameterDefinitions(path)

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Definition{
			{Name: "PORT", Description: "HTTP port", Required: true, Example: "8080"},
			{Name: "EMPTY"},
			{Name: "UNSET"},
		}, got)
	})

	t.Run("loads legacy args as parameters", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-topo:
  args:
    PORT: {}
`)

		got, err := project.LoadParameterDefinitions(path)

		require.NoError(t, err)
		assert.Equal(t, []parameter.Definition{{Name: "PORT"}}, got)
	})

	t.Run("prefers parameters over legacy args", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-topo:
  parameters:
    PORT: {}
  args:
    LEGACY: {}
`)

		got, err := project.LoadParameterDefinitions(path)

		require.NoError(t, err)
		assert.Equal(t, []parameter.Definition{{Name: "PORT"}}, got)
	})

	t.Run("resolves aliases and merge keys with explicit overrides", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-defaults: &defaults
  description: HTTP port
  required: true
  example: "8080"
x-topo:
  parameters:
    ORIGINAL: *defaults
    OVERRIDE:
      <<: *defaults
      example: "9000"
`)

		got, err := project.LoadParameterDefinitions(path)

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Definition{
			{Name: "ORIGINAL", Description: "HTTP port", Required: true, Example: "8080"},
			{Name: "OVERRIDE", Description: "HTTP port", Required: true, Example: "9000"},
		}, got)
	})

	t.Run("returns no definitions without metadata", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		got, err := project.LoadParameterDefinitions(path)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("reports missing files", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		got, err := project.LoadParameterDefinitions(path)

		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, got)
	})

	t.Run("reports invalid YAML", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo: [")

		got, err := project.LoadParameterDefinitions(path)

		assert.ErrorContains(t, err, "failed to decode project metadata")
		assert.Nil(t, got)
	})
}
