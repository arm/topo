package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadParameters(t *testing.T) {
	t.Run("loads parameter metadata", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-topo:
  parameters:
    PORT:
      description: HTTP port
      example: "8080"
    EMPTY: {}
    UNSET: {}
`)

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Parameter{
			{Name: "PORT", Description: "HTTP port", Example: "8080"},
			{Name: "EMPTY"},
			{Name: "UNSET"},
		}, got)
	})

	t.Run("loads references sorted by path and preserves duplicate expressions at distinct locations", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
    environment:
      FIRST: "${TOPO_TEST_PARAMETER:-one}/${TOPO_TEST_PARAMETER:-two}"
      SECOND: "${TOPO_TEST_PARAMETER:-one}/${TOPO_TEST_PARAMETER:-two}"
      ESCAPED: "$${TOPO_TEST_PARAMETER}"
    command: ["echo", "prefix-${TOPO_TEST_PARAMETER?required}"]
x-topo:
  parameters:
    TOPO_TEST_PARAMETER: {}
    TOPO_TEST_UNSET_PARAMETER: {}
`)

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Parameter{
			{
				Name: "TOPO_TEST_PARAMETER",
				References: []parameter.Reference{
					{Path: "services.app.command[1]", Expression: "prefix-${TOPO_TEST_PARAMETER?required}"},
					{Path: "services.app.environment.FIRST", Expression: "${TOPO_TEST_PARAMETER:-one}/${TOPO_TEST_PARAMETER:-two}"},
					{Path: "services.app.environment.SECOND", Expression: "${TOPO_TEST_PARAMETER:-one}/${TOPO_TEST_PARAMETER:-two}"},
				},
			},
			{Name: "TOPO_TEST_UNSET_PARAMETER"},
		}, got)
	})

	t.Run("loads current values from env files", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {TOPO_TEST_FILE_PARAMETER: "${TOPO_TEST_FILE_PARAMETER}"}
x-topo:
  parameters: {TOPO_TEST_FILE_PARAMETER: {}}
`)
		basePath := filepath.Join(root, ".env")
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, basePath, "TOPO_TEST_FILE_PARAMETER=base\n")
		testutil.RequireWriteFile(t, envPath, "TOPO_TEST_FILE_PARAMETER=configured\n")

		got, err := project.LoadParameters(project.Scope{ComposeFile: path, EnvFiles: []string{basePath, envPath}})

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{
			Name:          "TOPO_TEST_FILE_PARAMETER",
			ExistingValue: new("configured"),
			References:    []parameter.Reference{{Path: "services.app.environment.TOPO_TEST_FILE_PARAMETER", Expression: "${TOPO_TEST_FILE_PARAMETER}"}},
		}}, got)
	})

	t.Run("prefers process values over env file values", func(t *testing.T) {
		t.Setenv("TOPO_TEST_PARAMETER", "shell=value")
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {TOPO_TEST_PARAMETER: "${TOPO_TEST_PARAMETER}"}
x-topo:
  parameters: {TOPO_TEST_PARAMETER: {}}
`)
		basePath := filepath.Join(root, ".env")
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, basePath, "TOPO_TEST_PARAMETER=base\n")
		testutil.RequireWriteFile(t, envPath, "TOPO_TEST_PARAMETER=topo\n")

		got, err := project.LoadParameters(project.Scope{ComposeFile: path, EnvFiles: []string{basePath, envPath}})

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{
			Name:          "TOPO_TEST_PARAMETER",
			ExistingValue: new("shell=value"),
			References:    []parameter.Reference{{Path: "services.app.environment.TOPO_TEST_PARAMETER", Expression: "${TOPO_TEST_PARAMETER}"}},
		}}, got)
	})

	t.Run("does not read env files outside the input scope", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {TOPO_TEST_FILE_PARAMETER: "${TOPO_TEST_FILE_PARAMETER}"}
x-topo:
  parameters: {TOPO_TEST_FILE_PARAMETER: {}}
`)
		testutil.RequireWriteFile(t, filepath.Join(root, env.DefaultFilename), "invalid=\"unterminated\n")

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{
			Name:       "TOPO_TEST_FILE_PARAMETER",
			References: []parameter.Reference{{Path: "services.app.environment.TOPO_TEST_FILE_PARAMETER", Expression: "${TOPO_TEST_FILE_PARAMETER}"}},
		}}, got)
	})

	t.Run("loads legacy args as parameters", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-topo:
  args:
    PORT: {}
`)

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{Name: "PORT"}}, got)
	})

	t.Run("prefers parameters over legacy args", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-topo:
  parameters:
    PORT: {}
  args:
    LEGACY: {}
`)

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{Name: "PORT"}}, got)
	})

	t.Run("resolves aliases and merge keys with explicit overrides", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `x-defaults: &defaults
  description: HTTP port
  example: "8080"
x-topo:
  parameters:
    ORIGINAL: *defaults
    OVERRIDE:
      <<: *defaults
      example: "9000"
`)

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Parameter{
			{Name: "ORIGINAL", Description: "HTTP port", Example: "8080"},
			{Name: "OVERRIDE", Description: "HTTP port", Example: "9000"},
		}, got)
	})

	t.Run("returns no parameters without metadata", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("reports missing files", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, got)
	})

	t.Run("reports invalid YAML", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo: [")

		got, err := project.LoadParameters(project.Scope{ComposeFile: path})

		assert.ErrorContains(t, err, "failed to decode project metadata")
		assert.Nil(t, got)
	})
}
