package project_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

		err := project.Configure(path, filepath.Join(filepath.Dir(path), env.DefaultFilename), resolver)

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

		err := project.Configure(path, filepath.Join(filepath.Dir(path), env.DefaultFilename), resolver)

		require.NoError(t, err)
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
		testutil.RequireEnvFileValues(t, filepath.Join(filepath.Dir(path), env.DefaultFilename), map[string]string{"FOO": "baz"})
	})

	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		resolver := parameter.NewStrictResolverChain()

		err := project.Configure(invalidPath, filepath.Join(filepath.Dir(invalidPath), env.DefaultFilename), resolver)

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

		err := project.Configure(composeFilePath, filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename), resolver)

		require.NoError(t, err)
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
		testutil.RequireEnvFileValues(t, filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename), map[string]string{"FOO": "baz"})
	})
}
