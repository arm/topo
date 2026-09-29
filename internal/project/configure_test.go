package project_test

import (
	"bytes"
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

func TestConfigure(t *testing.T) {
	t.Run("process values satisfy required parameters without returning updates", func(t *testing.T) {
		t.Setenv("TOPO_TEST_PARAMETER", "shell=value")
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {TOPO_TEST_PARAMETER: "${TOPO_TEST_PARAMETER}"}
x-topo:
  parameters: {TOPO_TEST_PARAMETER: {required: true}}
`)

		values, err := project.Configure(project.Scope{ComposeFile: path}, parameter.NewStrictResolverChain())

		require.NoError(t, err)
		assert.Nil(t, values)
		assert.NoFileExists(t, filepath.Join(root, env.DefaultFilename))
	})

	t.Run("interactive current values prefer process values without returning updates", func(t *testing.T) {
		t.Setenv("TOPO_TEST_PARAMETER", "shell=value")
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {TOPO_TEST_PARAMETER: "${TOPO_TEST_PARAMETER}"}
x-topo:
  parameters: {TOPO_TEST_PARAMETER: {required: true}}
`)
		basePath := filepath.Join(root, ".env")
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, basePath, "TOPO_TEST_PARAMETER=base\n")
		testutil.RequireWriteFile(t, envPath, "TOPO_TEST_PARAMETER=topo\n")
		scope := project.Scope{ComposeFile: path, EnvFiles: []string{basePath, envPath}}
		output := &bytes.Buffer{}
		resolver := parameter.NewStrictResolverChain(parameter.NewInteractiveResolver(strings.NewReader("\n"), output))

		values, err := project.Configure(scope, resolver)

		require.NoError(t, err)
		assert.Nil(t, values)
		assert.Contains(t, output.String(), `Current: "shell=value"`)
		testutil.RequireEnvFileValues(t, envPath, map[string]string{"TOPO_TEST_PARAMETER": "topo"})
	})

	t.Run("uses inherited values without returning them as updates", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {A: "${A}", B: "${B}"}
x-topo:
  parameters: {A: {}, B: {required: true}}
`)
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, envPath, "A=original\n")
		basePath := filepath.Join(root, ".env")
		testutil.RequireWriteFile(t, basePath, "B=keep-me\n")
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"A": "updated"}))

		values, err := project.Configure(project.Scope{ComposeFile: path, EnvFiles: []string{basePath, envPath}}, resolver)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"A": "updated"}, values)
	})

	t.Run("does not read env files outside the input scope", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    image: alpine
    environment: {A: "${A}"}
x-topo:
  parameters: {A: {}}
`)
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, envPath, "invalid=\"unterminated\n")
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(parameter.Values{"A": "updated"}))

		values, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"A": "updated"}, values)
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

		values, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"FOO": "baz"}, values)
	})

	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		resolver := parameter.NewStrictResolverChain()

		values, err := project.Configure(project.Scope{ComposeFile: invalidPath}, resolver)

		require.ErrorContains(t, err, "failed to open compose file")
		assert.Nil(t, values)
	})

	t.Run("returns provided parameters overriding process values and leaves Compose unchanged", func(t *testing.T) {
		t.Setenv("TOPO_TEST_PARAMETER", "shell")
		composeFileContents := `
services:
  app:
    build:
      context: .
      args:
        FOO: ${TOPO_TEST_PARAMETER}

x-topo:
  name: My Project
  parameters:
    TOPO_TEST_PARAMETER:
      description: a dummy parameter
      required: true
      example: bar
`
		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), composeFileContents)
		static := parameter.NewStaticResolver(parameter.Values{"TOPO_TEST_PARAMETER": "baz"})
		resolver := parameter.NewStrictResolverChain(static)

		values, err := project.Configure(project.Scope{ComposeFile: composeFilePath}, resolver)

		require.NoError(t, err)
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
		assert.Equal(t, map[string]string{"TOPO_TEST_PARAMETER": "baz"}, values)
	})
}
