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
	const composeWithInterpolatedInclude = `include:
  - ${TOPO_TEST_INCLUDE_FILE}
x-topo:
  parameters:
    TOPO_TEST_GREETING: {}
`
	const includedComposeWithRequiredGreeting = `services:
  app:
    image: alpine
    environment:
      GREETING: ${TOPO_TEST_GREETING:?Greeting is required}
`

	t.Run("accepts unreferenced Topo parameters", func(t *testing.T) {
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

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

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

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("does not load extended files for projects without Topo parameters", func(t *testing.T) {
		t.Setenv("BASE_FILE", "")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends:
      file: "${BASE_FILE:-base.yaml}"
      service: common
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("resolves a required Topo parameter in an interpolated include", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), includedComposeWithRequiredGreeting)
		t.Setenv("TOPO_TEST_INCLUDE_FILE", included)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), composeWithInterpolatedInclude)
		changes := parameter.Changes{"TOPO_TEST_GREETING": new("hello")}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("rejects a missing required Topo parameter in an interpolated include", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), includedComposeWithRequiredGreeting)
		t.Setenv("TOPO_TEST_INCLUDE_FILE", included)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), composeWithInterpolatedInclude)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, "required variable TOPO_TEST_GREETING")
		assert.Nil(t, got)
	})

	t.Run("accepts a required Topo parameter supplied by the included project's dotenv file", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_GREETING")
		root := t.TempDir()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		testutil.RequireWriteComposeFile(t, includedDirectory, includedComposeWithRequiredGreeting)
		testutil.RequireWriteFile(t, filepath.Join(includedDirectory, ".env"), "TOPO_TEST_GREETING=hello\n")
		path := testutil.RequireWriteComposeFile(t, root, `include:
  - ./included/compose.yaml
x-topo:
  parameters:
    TOPO_TEST_GREETING: {}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("rejects an explicit empty value overriding an included dotenv default", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_GREETING")
		root := t.TempDir()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		testutil.RequireWriteComposeFile(t, includedDirectory, `services:
  app:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
`)
		testutil.RequireWriteFile(t, filepath.Join(includedDirectory, ".env"), "TOPO_TEST_GREETING=hello\n")
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		changes := parameter.Changes{"TOPO_TEST_GREETING": new("")}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, "required variable TOPO_TEST_GREETING")
		assert.Nil(t, got)
	})

	t.Run("does not use an included dotenv value to satisfy a required root reference", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_GREETING")
		root := t.TempDir()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		testutil.RequireWriteComposeFile(t, includedDirectory, `services:
  included:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
`)
		testutil.RequireWriteFile(t, filepath.Join(includedDirectory, ".env"), "TOPO_TEST_GREETING=hello\n")
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
services:
  root:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, "required variable TOPO_TEST_GREETING")
		assert.Nil(t, got)
	})
}

func TestConfigureLiteralExtends(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		value     string
		wantError bool
	}{
		{name: "accepts the calling project's environment", value: "hello"},
		{name: "rejects a missing value despite dotenv beside the extended file", wantError: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			testutil.Unsetenv(t, "TOPO_TEST_EXTENDS_GREETING")
			root := t.TempDir()
			baseDirectory := filepath.Join(root, "base")
			testutil.RequireMkdirAll(t, baseDirectory)
			testutil.RequireWriteComposeFile(t, baseDirectory, `services:
  common:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_EXTENDS_GREETING:?Greeting is required}"}
`)
			testutil.RequireWriteFile(t, filepath.Join(baseDirectory, ".env"), "TOPO_TEST_EXTENDS_GREETING=ignored\n")
			path := testutil.RequireWriteComposeFile(t, root, `services:
  app:
    extends: {file: ./base/compose.yaml, service: common}
x-topo:
  parameters: {TOPO_TEST_EXTENDS_GREETING: {}}
`)
			if scenario.value != "" {
				t.Setenv("TOPO_TEST_EXTENDS_GREETING", scenario.value)
			}
			resolver := parameter.NewStrictResolverChain()

			got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

			if scenario.wantError {
				require.ErrorContains(t, err, "TOPO_TEST_EXTENDS_GREETING")
			} else {
				require.NoError(t, err)
			}
			assert.Empty(t, got)
		})
	}
}
