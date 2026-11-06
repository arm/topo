package project_test

import (
	"testing"

	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	t.Run("accepts an interpolated include path when there are no parameters", func(t *testing.T) {
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
`)
		t.Setenv("TOPO_TEST_INCLUDE_FILE", included)
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_INCLUDE_FILE}
`)
		override := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    command: echo hello
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFiles: []string{base, override}}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("skips referenced files when there are no parameters", func(t *testing.T) {
		t.Setenv("BASE_FILE", "")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends:
      file: "${BASE_FILE:-base.yaml}"
      service: common
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFiles: []string{path}}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("rejects missing required value even when override supplies a default", func(t *testing.T) {
		t.Setenv("VERSION", "")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine:${VERSION:?VERSION is required}
x-topo:
  parameters:
    VERSION: {}
`)
		override := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine:${VERSION:-latest}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFiles: []string{base, override}}, resolver)

		require.ErrorContains(t, err, "VERSION")
		assert.Nil(t, got)
	})

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
}
