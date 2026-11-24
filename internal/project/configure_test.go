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

func TestConfigure(t *testing.T) {
	t.Run("accepts unreferenced parameters", func(t *testing.T) {
		contents := `services:
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
		changes := parameter.Changes{"FOO": new("baz")}
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), contents)
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFiles: []string{path}}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("returns resolver updates", func(t *testing.T) {
		contents := `services:
  app:
    image: alpine
    environment: {GREETING: "${GREETING:-hello}"}
x-topo:
  parameters: {GREETING: {}}
`
		changes := parameter.Changes{"GREETING": new("updated")}
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), contents)
		resolver := parameter.NewStaticResolver(changes)

		got, err := project.Configure(project.Scope{ComposeFiles: []string{path}}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("only accepts a required token from an included .env.topo if Compose can load it", func(t *testing.T) {
		const key = "TOPO_TEST_INCLUDED_TOPO_TOKEN"
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
		childDir := t.TempDir()
		child := testutil.RequireWriteComposeFile(t, childDir, `services:
  app:
    image: alpine
    environment: {TOKEN: "${TOPO_TEST_INCLUDED_TOPO_TOKEN:?token is required}"}
`)
		testutil.RequireWriteFile(t, filepath.Join(childDir, env.DefaultFilename), key+"=only_in_child_topo_env\n")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - `+child+`
x-topo:
  parameters: {TOPO_TEST_INCLUDED_TOPO_TOKEN: {}}
`)
		scope, err := project.BuildScope([]string{path}, "", nil)
		require.NoError(t, err)

		_, configureErr := project.Configure(scope, parameter.NewStrictResolverChain())
		_, readErr := project.Read(scope)
		if configureErr == nil {
			require.NoError(t, readErr, "configuration accepted a token that Compose could not resolve")
			return
		}
		var validationErrors parameter.ValidationErrors
		require.ErrorAs(t, configureErr, &validationErrors)
		assert.ErrorContains(t, configureErr, key)
		require.ErrorContains(t, readErr, "token is required")
	})
}
