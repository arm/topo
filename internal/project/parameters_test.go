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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

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

		scope, err := project.BuildScope([]string{path}, "", nil)
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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}, EnvFiles: []string{basePath, envPath}})

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

		scope, err := project.BuildScope([]string{path}, "", nil)
		require.NoError(t, err)

		got, err := project.LoadParameters(scope)

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{
			Name:          "TOPO_TEST_FILE_PARAMETER",
			ExistingValue: new("value_from_sibling"),
			References:    []parameter.Reference{{Path: "services.app.environment.TOPO_TEST_FILE_PARAMETER", Expression: "${TOPO_TEST_FILE_PARAMETER}"}},
		}}, got)
	})

	t.Run("adjacent .env.topo overrides .env", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		require.NoError(t, os.Unsetenv("TOPO_TEST_GREETING"))
		directory := t.TempDir()
		contents := "services:\n  app:\n    image: alpine\n    environment: {GREETING: '${TOPO_TEST_GREETING}'}\nx-topo:\n  parameters: {TOPO_TEST_GREETING: {}}\n"
		composeFile := testutil.RequireWriteComposeFile(t, directory, contents)
		testutil.RequireWriteFile(t, filepath.Join(directory, ".env"), "TOPO_TEST_GREETING=default\n")
		testutil.RequireWriteFile(t, filepath.Join(directory, ".env.topo"), "TOPO_TEST_GREETING=configured\n")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{composeFile}})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, new("configured"), got[0].ExistingValue)
	})

	t.Run("later Compose file environment overrides earlier files", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		require.NoError(t, os.Unsetenv("TOPO_TEST_GREETING"))
		baseDirectory, overrideDirectory := t.TempDir(), t.TempDir()
		contents := "services:\n  app:\n    image: alpine\n    environment: {GREETING: '${TOPO_TEST_GREETING}'}\nx-topo:\n  parameters: {TOPO_TEST_GREETING: {}}\n"
		base := testutil.RequireWriteComposeFile(t, baseDirectory, contents)
		override := testutil.RequireWriteComposeFile(t, overrideDirectory, "services: {}\n")
		testutil.RequireWriteFile(t, filepath.Join(baseDirectory, ".env"), "TOPO_TEST_GREETING=base\n")
		testutil.RequireWriteFile(t, filepath.Join(overrideDirectory, ".env"), "TOPO_TEST_GREETING=override\n")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{base, override}})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, new("override"), got[0].ExistingValue)
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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}, EnvFiles: []string{basePath, envPath}})

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
		testutil.RequireWriteFile(t, filepath.Join(t.TempDir(), env.DefaultFilename), "invalid=\"unterminated\n")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

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

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

		require.NoError(t, err)
		assert.ElementsMatch(t, []parameter.Parameter{
			{Name: "ORIGINAL", Description: "HTTP port", Example: "8080"},
			{Name: "OVERRIDE", Description: "HTTP port", Example: "9000"},
		}, got)
	})

	t.Run("returns no parameters without metadata", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("reports missing files", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

		assert.ErrorIs(t, err, os.ErrNotExist)
		assert.Nil(t, got)
	})

	t.Run("reports invalid YAML", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo: [")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{path}})

		assert.ErrorContains(t, err, "failed to decode project metadata")
		assert.Nil(t, got)
	})

	t.Run("collects parameter declarations from every Compose file", func(t *testing.T) {
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo:\n  parameters:\n    FIRST: {description: First parameter}\n")
		override := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo:\n  parameters:\n    SECOND: {description: Second parameter}\n")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{base, override}})

		require.NoError(t, err)
		var declarations []parameter.Parameter
		for _, declared := range got {
			declarations = append(declarations, parameter.Parameter{Name: declared.Name, Description: declared.Description})
		}
		assert.ElementsMatch(t, []parameter.Parameter{
			{Name: "FIRST", Description: "First parameter"},
			{Name: "SECOND", Description: "Second parameter"},
		}, declarations)
	})

	t.Run("later Compose files override duplicate parameter declarations", func(t *testing.T) {
		t.Setenv("TOPO_TEST_PARAMETER", "")
		require.NoError(t, os.Unsetenv("TOPO_TEST_PARAMETER"))
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo:\n  parameters:\n    TOPO_TEST_PARAMETER: {description: Original}\n")
		override := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo:\n  parameters:\n    TOPO_TEST_PARAMETER: {description: Updated}\n")

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{base, override}})

		require.NoError(t, err)
		assert.Equal(t, []parameter.Parameter{{Name: "TOPO_TEST_PARAMETER", Description: "Updated"}}, got)
	})

	t.Run("discovers parameter references in later Compose files", func(t *testing.T) {
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		override := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    environment:
      GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"
`)

		got, err := project.LoadParameters(project.Scope{ComposeFiles: []string{base, override}})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, []parameter.Reference{{
			Path:       "services.app.environment.GREETING",
			Expression: "${TOPO_TEST_GREETING:?Greeting is required}",
		}}, got[0].References)
	})
}
