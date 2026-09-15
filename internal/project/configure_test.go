package project_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
