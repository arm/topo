package project_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateToEnv(t *testing.T) {
	const contents = `services:
  app:
    build:
      args:
        FOO: current
        OTHER: unchanged
x-topo:
  parameters:
    FOO: {}
`

	t.Run("moves current values to env and replaces only declared parameters", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := strings.Replace(contents, "FOO: current", "FOO: ${FOO?configured via topo}", 1)

		err := project.MigrateToEnv(path)

		require.NoError(t, err)
		envContents := testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename))
		assert.Contains(t, envContents, "\nFOO=\"current\"\n")
		assert.NotContains(t, envContents, "OTHER=")
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves both files when env already exists", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, contents)
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, envPath, "FOO=existing\n")

		err := project.MigrateToEnv(path)

		require.ErrorContains(t, err, "env file already exists")
		assert.Equal(t, "FOO=existing\n", testutil.RequireReadFile(t, envPath))
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
	})

	t.Run("skips declared parameters with no matching build arg", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        PRESENT: ""
x-topo:
  parameters:
    MISSING: {}
    PRESENT: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := `services:
  app:
    build:
      args:
        PRESENT: ${PRESENT?configured via topo}
x-topo:
  parameters:
    MISSING: {}
    PRESENT: {}
`

		err := project.MigrateToEnv(path)

		require.NoError(t, err)
		envContents := testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename))
		assert.NotContains(t, envContents, "MISSING=")
		assert.Contains(t, envContents, "\nPRESENT=\"\"\n")
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("rejects parameters with multiple current values", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  first:
    build:
      args:
        FOO: current
  second:
    build:
      args:
        FOO: another
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)

		err := project.MigrateToEnv(path)

		require.ErrorContains(t, err, "parameter FOO has more than one current value")
		assert.NoFileExists(t, filepath.Join(root, env.DefaultFilename))
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
	})
}

func TestConfigure(t *testing.T) {
	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		resolver := parameter.NewStrictResolverChain()

		err := project.Configure(invalidPath, resolver)

		require.ErrorContains(t, err, "can't read compose file")
	})

	t.Run("updates the compose file with provided parameters", func(t *testing.T) {
		composeFileContents := `
services:
  app:
    build:
      context: .
      args:
        FOO: bar

x-topo:
  name: My Project
  parameters:
    FOO:
      description: a dummy parameter
      required: true
      example: bar
`
		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), composeFileContents)
		static := parameter.NewStaticResolver(parameter.Values{"FOO": "baz"})
		resolver := parameter.NewStrictResolverChain(static)

		err := project.Configure(composeFilePath, resolver)
		require.NoError(t, err)

		want := `
services:
  app:
    build:
      context: .
      args:
        FOO: baz

x-topo:
  name: My Project
  parameters:
    FOO:
      description: a dummy parameter
      required: true
      example: bar
`
		got := testutil.RequireReadFile(t, composeFilePath)

		assert.YAMLEq(t, want, got)
	})

	t.Run("rejects empty input for required parameters when any current value is empty", func(t *testing.T) {
		composeFileContents := `services:
  configured:
    build:
      args:
        FOO: current
  empty:
    build:
      args:
        FOO: ""
x-topo:
  parameters:
    FOO:
      required: true
      default: default
`
		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), composeFileContents)
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(nil))

		err := project.Configure(composeFilePath, resolver)

		require.ErrorContains(t, err, "missing value(s) for required parameters")
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
	})

	t.Run("recognizes current values in sequence build args", func(t *testing.T) {
		composeFileContents := `services:
  app:
    build:
      args: ["FOO=current"]
x-topo:
  parameters:
    FOO:
      required: true
      default: default
`
		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), composeFileContents)
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(nil))

		err := project.Configure(composeFilePath, resolver)

		require.NoError(t, err)
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
	})
}
