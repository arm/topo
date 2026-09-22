package project_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsesLiteralBuildArgConfiguration(t *testing.T) {
	t.Run("detects legacy build argument values without migrating the project", func(t *testing.T) {
		path := writeGreetingComposeFile(t, "World")
		original := testutil.RequireReadFile(t, path)

		usesLiteralBuildArgs, err := project.UsesLiteralBuildArgConfiguration(path)

		require.NoError(t, err)
		assert.True(t, usesLiteralBuildArgs)
		assert.Equal(t, original, testutil.RequireReadFile(t, path))
		assert.NoFileExists(t, filepath.Join(filepath.Dir(path), env.DefaultFilename))
	})

	t.Run("does not classify environment-backed build arguments as legacy", func(t *testing.T) {
		path := writeGreetingComposeFile(t, "${GREETING_NAME?configured via topo}")

		usesLiteralBuildArgs, err := project.UsesLiteralBuildArgConfiguration(path)

		require.NoError(t, err)
		assert.False(t, usesLiteralBuildArgs)
	})

	t.Run("propagates inspection errors", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		usesLiteralBuildArgs, err := project.UsesLiteralBuildArgConfiguration(path)

		require.Error(t, err)
		assert.False(t, usesLiteralBuildArgs)
	})
}

func writeGreetingComposeFile(t *testing.T, greeting string) string {
	t.Helper()
	return testutil.RequireWriteComposeFile(t, t.TempDir(), fmt.Sprintf(`services:
  app:
    build:
      args:
        GREETING_NAME: %q
x-topo:
  parameters:
    GREETING_NAME: {}
`, greeting))
}

func TestMigrateToEnv(t *testing.T) {
	t.Run("migrates repeated identical values and replaces only declared parameters", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: current
        OTHER: unchanged
  second:
    build:
      args: ["FOO=current"]
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := `services:
  app:
    build:
      args:
        FOO: ${FOO?configured via topo}
        OTHER: unchanged
  second:
    build:
      args: ["FOO=${FOO?configured via topo}"]
x-topo:
  parameters:
    FOO: {}
`

		err := project.MigrateToEnv(path)

		require.NoError(t, err)
		envContents := testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename))
		assert.Contains(t, envContents, "\nFOO=\"current\"\n")
		assert.NotContains(t, envContents, "OTHER=")
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves build arg interpolation in the env file", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: ${FOO}
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := `services:
  app:
    build:
      args:
        FOO: ${FOO?configured via topo}
x-topo:
  parameters:
    FOO: {}
`

		err := project.MigrateToEnv(path)

		require.NoError(t, err)
		assert.Contains(t, testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename)), "\nFOO=\"${FOO}\"\n")
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves files when there are no parameter values to migrate", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: current
x-topo:
  parameters:
    MISSING: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)

		err := project.MigrateToEnv(path)

		require.EqualError(t, err, "no parameter values to migrate; only projects with referenced parameters can be migrated")
		assert.NoFileExists(t, filepath.Join(root, env.DefaultFilename))
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves both files when env already exists", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: current
x-topo:
  parameters:
    FOO: {}
`
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
	t.Run("preserves existing values when updating one parameter", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {A: "${A}", B: "${B}"}
x-topo:
  parameters: {A: {}, B: {required: true}}
`)
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, envPath, "A=original\nB=keep-me\n")
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"A": "updated"}))

		err := project.Configure(path, resolver)

		require.NoError(t, err)
		testutil.RequireEnvFileValues(t, envPath, map[string]string{"A": "updated", "B": "keep-me"})
	})

	t.Run("allows unreferenced parameters with no matching build arg", func(t *testing.T) {
		contents := `
services:
  app:
    platform: linux/arm64
    build:
      context: .
      args:
        TEST: value
x-topo:
  deployment_success_message: "Access it at http://${TOPO_TARGET_HOSTNAME}:8080"
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), contents)
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"FOO": "baz"}))

		err := project.Configure(path, resolver)

		require.NoError(t, err)
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
		testutil.RequireEnvFileValues(t, filepath.Join(filepath.Dir(path), env.DefaultFilename), map[string]string{"FOO": "baz"})
	})

	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		resolver := parameter.NewStrictResolverChain()

		err := project.Configure(invalidPath, resolver)

		require.ErrorContains(t, err, "failed to open compose file")
	})

	t.Run("writes provided parameters to env and leaves Compose unchanged", func(t *testing.T) {
		composeFileContents := `
services:
  app:
    build:
      context: .
      args:
        FOO: ${FOO}

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
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
		testutil.RequireEnvFileValues(t, filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename), map[string]string{"FOO": "baz"})
	})
}
