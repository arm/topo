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
      source: {provider: test, group: settings, fact: port}
    EMPTY: {}
    UNSET: {}
`)
		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Parameter{
			{Name: "PORT", Description: "HTTP port", Example: "8080", Source: &parameter.Source{Provider: "test", Group: "settings", Fact: "port"}},
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

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

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

	t.Run("loads parameter references from a long-form Compose include", func(t *testing.T) {
		child := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
    environment: {TOKEN: "${TOPO_TEST_LONG_FORM_TOKEN}"}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - path: `+child+`
x-topo:
  parameters: {TOPO_TEST_LONG_FORM_TOKEN: {}}
`)

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, []parameter.Reference{{
			Path:       "services.app.environment.TOKEN",
			Expression: "${TOPO_TEST_LONG_FORM_TOKEN}",
		}}, got[0].References)
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

		scope, err := project.BuildScope(path, "", []string{basePath, envPath})
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{
			Name:          "TOPO_TEST_FILE_PARAMETER",
			ExistingValue: new("configured"),
			References:    []parameter.Reference{{Path: "services.app.environment.TOPO_TEST_FILE_PARAMETER", Expression: "${TOPO_TEST_FILE_PARAMETER}"}},
		}}, got)
	})

	t.Run("loads current values from .env beside an included Compose file", func(t *testing.T) {
		t.Setenv("TOPO_TEST_FILE_PARAMETER", "")
		require.NoError(t, os.Unsetenv("TOPO_TEST_FILE_PARAMETER"))
		includeFolder := t.TempDir()
		baseFolder := t.TempDir()
		includeComposeFile := testutil.RequireWriteComposeFile(t, includeFolder, `services:
  app:
    image: alpine
    environment: {TOPO_TEST_FILE_PARAMETER: "${TOPO_TEST_FILE_PARAMETER}"}
`)
		path := testutil.RequireWriteComposeFile(t, baseFolder, `include:
  - `+includeComposeFile+`

x-topo:
  parameters: {TOPO_TEST_FILE_PARAMETER: {}}
`)
		includeSiblingEnv := filepath.Join(includeFolder, ".env")
		testutil.RequireWriteFile(t, includeSiblingEnv, "TOPO_TEST_FILE_PARAMETER=value_from_sibling\n")

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{
			Name:          "TOPO_TEST_FILE_PARAMETER",
			ExistingValue: new("value_from_sibling"),
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

		scope, err := project.BuildScope(path, "", []string{basePath, envPath})
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

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

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

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

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

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

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

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

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Parameter{
			{Name: "ORIGINAL", Description: "HTTP port", Example: "8080"},
			{Name: "OVERRIDE", Description: "HTTP port", Example: "9000"},
		}, got)
	})

	t.Run("returns no parameters without metadata", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		scope, err := project.BuildScope(path, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
