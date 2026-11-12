package project_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/compose-spec/compose-go/v2/template"
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
	writeProjectWithInterpolatedInclude := func(t *testing.T) string {
		t.Helper()
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), includedComposeWithRequiredGreeting)
		t.Setenv("TOPO_TEST_INCLUDE_FILE", included)
		return testutil.RequireWriteComposeFile(t, t.TempDir(), composeWithInterpolatedInclude)
	}
	writeIncludedProjectWithDotenv := func(t *testing.T, root, contents string) {
		t.Helper()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		testutil.RequireWriteComposeFile(t, includedDirectory, contents)
		testutil.RequireWriteFile(t, filepath.Join(includedDirectory, ".env"), "TOPO_TEST_GREETING=hello\n")
	}

	t.Run("resolves a nested extends path through a sibling service in the inherited file", func(t *testing.T) {
		t.Setenv("TOPO_TEST_BASE_FILE", "")
		t.Setenv("TOPO_TEST_COMMON_FILE", "")
		common := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {common: {image: alpine}}\n")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  base:
    extends: sibling
  sibling:
    extends: {file: "${TOPO_TEST_COMMON_FILE:?Choose common}", service: common}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends: {file: "${TOPO_TEST_BASE_FILE:?Choose base}", service: base}
x-topo:
  parameters: {TOPO_TEST_BASE_FILE: {}, TOPO_TEST_COMMON_FILE: {}}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{
			"TOPO_TEST_BASE_FILE=" + base, "TOPO_TEST_COMMON_FILE=" + common,
		}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"TOPO_TEST_BASE_FILE": new(base), "TOPO_TEST_COMMON_FILE": new(common)}, got)
	})

	t.Run("rejects a cycle formed by resolved extends path parameters", func(t *testing.T) {
		t.Setenv("TOPO_TEST_BASE_FILE", "")
		t.Setenv("TOPO_TEST_ROOT_FILE", "")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  base:
    extends: {file: "${TOPO_TEST_ROOT_FILE:?Choose root}", service: app}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends: {file: "${TOPO_TEST_BASE_FILE:?Choose base}", service: base}
x-topo:
  parameters: {TOPO_TEST_BASE_FILE: {}, TOPO_TEST_ROOT_FILE: {}}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{
			"TOPO_TEST_BASE_FILE=" + base,
			"TOPO_TEST_ROOT_FILE=" + path,
		}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, "extends cycle detected")
		assert.Nil(t, got)
	})

	t.Run("resolves nested extends paths inside a parameterized include", func(t *testing.T) {
		t.Setenv("TOPO_TEST_CONFIG_FILE", "")
		t.Setenv("TOPO_TEST_BASE_FILE", "")
		t.Setenv("TOPO_TEST_COMMON_FILE", "")
		common := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {common: {image: alpine}}\n")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  base:
    extends: {file: "${TOPO_TEST_COMMON_FILE:?Choose common}", service: common}
`)
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends: {file: "${TOPO_TEST_BASE_FILE:?Choose base}", service: base}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters: {TOPO_TEST_CONFIG_FILE: {}, TOPO_TEST_BASE_FILE: {}, TOPO_TEST_COMMON_FILE: {}}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_CONFIG_FILE=" + included, "TOPO_TEST_BASE_FILE=" + base, "TOPO_TEST_COMMON_FILE=" + common}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"TOPO_TEST_CONFIG_FILE": new(included), "TOPO_TEST_BASE_FILE": new(base), "TOPO_TEST_COMMON_FILE": new(common)}, got)
	})

	t.Run("resolves a parameterized extends path discovered in an inherited service", func(t *testing.T) {
		t.Setenv("TOPO_TEST_BASE_FILE", "")
		t.Setenv("TOPO_TEST_COMMON_FILE", "")
		common := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {common: {image: alpine}}\n")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  base:
    extends:
      file: ${TOPO_TEST_COMMON_FILE:?Choose a common configuration}
      service: common
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends:
      file: ${TOPO_TEST_BASE_FILE:?Choose a base configuration}
      service: base
x-topo:
  parameters: {TOPO_TEST_BASE_FILE: {}, TOPO_TEST_COMMON_FILE: {}}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_BASE_FILE=" + base, "TOPO_TEST_COMMON_FILE=" + common}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"TOPO_TEST_BASE_FILE": new(base), "TOPO_TEST_COMMON_FILE": new(common)}, got)
	})

	t.Run("stops when the resolver leaves a required include path unresolved", func(t *testing.T) {
		t.Setenv("TOPO_TEST_CONFIG_FILE", "")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters:
    TOPO_TEST_CONFIG_FILE: {}
`)
		promptCount := 0
		interactive := parameter.NewInteractiveResolver(func(term.Prompt) (string, error) {
			promptCount++
			if promptCount > 1 {
				t.Fatal("retried an unresolved path parameter")
			}
			return "", nil
		}, term.NewPalette(false))
		resolver := parameter.NewStrictResolverChain(interactive)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		var missing *template.MissingRequiredError
		require.ErrorAs(t, err, &missing)
		assert.Equal(t, "TOPO_TEST_CONFIG_FILE", missing.Variable)
		assert.Equal(t, 1, promptCount)
		assert.Nil(t, got)
	})

	t.Run("rejects a cycle formed by resolved include path parameters", func(t *testing.T) {
		t.Setenv("TOPO_TEST_CONFIG_FILE", "")
		t.Setenv("TOPO_TEST_ROOT_FILE", "")
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_ROOT_FILE:?Choose the root configuration}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters:
    TOPO_TEST_CONFIG_FILE: {}
    TOPO_TEST_ROOT_FILE: {}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_CONFIG_FILE=" + included, "TOPO_TEST_ROOT_FILE=" + path}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, "include cycle detected")
		assert.Nil(t, got)
	})

	t.Run("resolves a parameterized include path discovered inside another include", func(t *testing.T) {
		t.Setenv("TOPO_TEST_CONFIG_FILE", "")
		t.Setenv("TOPO_TEST_DATABASE_CONFIG", "")
		database := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {database: {image: postgres}}\n")
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_DATABASE_CONFIG:?Choose a database configuration}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters:
    TOPO_TEST_CONFIG_FILE: {}
    TOPO_TEST_DATABASE_CONFIG: {}
`)
		changes := parameter.Changes{
			"TOPO_TEST_CONFIG_FILE": new(included), "TOPO_TEST_DATABASE_CONFIG": new(database),
		}
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_CONFIG_FILE=" + included, "TOPO_TEST_DATABASE_CONFIG=" + database}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("prompts only once when an existing include path is accepted unchanged", func(t *testing.T) {
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {app: {image: alpine}}\n")
		t.Setenv("TOPO_TEST_CONFIG_FILE", included)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters:
    TOPO_TEST_CONFIG_FILE: {}
`)
		var promptedValues []string
		interactive := parameter.NewInteractiveResolver(func(prompt term.Prompt) (string, error) {
			promptedValues = append(promptedValues, prompt.Initial)
			return prompt.Initial, nil
		}, term.NewPalette(false))
		resolver := parameter.NewStrictResolverChain(interactive)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, []string{included}, promptedValues)
		assert.Nil(t, got)
	})

	t.Run("accepts CLI values for both an include path and an included service parameter", func(t *testing.T) {
		t.Setenv("TOPO_TEST_CONFIG_FILE", "")
		t.Setenv("TOPO_TEST_GREETING", "")
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters:
    TOPO_TEST_CONFIG_FILE: {}
    TOPO_TEST_GREETING: {}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_CONFIG_FILE=" + included, "TOPO_TEST_GREETING=hello"}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"TOPO_TEST_CONFIG_FILE": new(included), "TOPO_TEST_GREETING": new("hello")}, got)
	})

	t.Run("resolves an include path supplied by a Topo parameter resolver", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_CONFIG_FILE")
		included := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
`)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `include:
  - ${TOPO_TEST_CONFIG_FILE:?Choose a configuration file}
x-topo:
  parameters:
    TOPO_TEST_CONFIG_FILE: {}
`)
		changes := parameter.Changes{"TOPO_TEST_CONFIG_FILE": new(included)}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

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

	t.Run("accepts an unset required Topo parameter inside an unused fallback", func(t *testing.T) {
		t.Setenv("TOPO_TEST_OUTER", "hello")
		testutil.Unsetenv(t, "TOPO_TEST_INNER")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    image: alpine
    environment:
      GREETING: ${TOPO_TEST_OUTER:-${TOPO_TEST_INNER:?Inner is required}}
x-topo:
  parameters:
    TOPO_TEST_INNER: {}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
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

	t.Run("resolves a Topo parameter declared only in an included file", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_GREETING")
		root := t.TempDir()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		testutil.RequireWriteComposeFile(t, includedDirectory, `services:
  app:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		path := testutil.RequireWriteComposeFile(t, root, "include: [./included/compose.yaml]\n")
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_GREETING=hello"}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"TOPO_TEST_GREETING": new("hello")}, got)
	})

	t.Run("rejects duplicate Topo parameter declarations in root and included files", func(t *testing.T) {
		root := t.TempDir()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		included := testutil.RequireWriteComposeFile(t, includedDirectory, `services:
  app:
    image: alpine
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, `duplicate Topo parameter "TOPO_TEST_GREETING"`)
		assert.Contains(t, err.Error(), included)
		assert.Nil(t, got)
	})

	t.Run("resolves a nested include path declared by the included file", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_DATABASE_CONFIG")
		root := t.TempDir()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		database := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {database: {image: postgres}}\n")
		testutil.RequireWriteComposeFile(t, includedDirectory, `include:
  - ${TOPO_TEST_DATABASE_CONFIG:?Choose a database configuration}
x-topo:
  parameters:
    TOPO_TEST_DATABASE_CONFIG: {}
`)
		path := testutil.RequireWriteComposeFile(t, root, "include: [./included/compose.yaml]\n")
		cliResolver, err := parameter.NewCLIResolver([]string{"TOPO_TEST_DATABASE_CONFIG=" + database}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"TOPO_TEST_DATABASE_CONFIG": new(database)}, got)
	})

	t.Run("resolves a required Topo parameter in an interpolated include", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		path := writeProjectWithInterpolatedInclude(t)
		changes := parameter.Changes{"TOPO_TEST_GREETING": new("hello")}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("rejects a missing required Topo parameter in an interpolated include", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		path := writeProjectWithInterpolatedInclude(t)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		var missing *template.MissingRequiredError
		require.ErrorAs(t, err, &missing)
		assert.Equal(t, "TOPO_TEST_GREETING", missing.Variable)
		assert.Nil(t, got)
	})

	t.Run("accepts a required Topo parameter supplied by the included project's dotenv file", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_GREETING")
		root := t.TempDir()
		writeIncludedProjectWithDotenv(t, root, includedComposeWithRequiredGreeting)
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
		writeIncludedProjectWithDotenv(t, root, includedComposeWithRequiredGreeting)
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		changes := parameter.Changes{"TOPO_TEST_GREETING": new("")}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		var missing *template.MissingRequiredError
		require.ErrorAs(t, err, &missing)
		assert.Equal(t, "TOPO_TEST_GREETING", missing.Variable)
		assert.Nil(t, got)
	})

	t.Run("does not use an included dotenv value to satisfy a required root reference", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_GREETING")
		root := t.TempDir()
		writeIncludedProjectWithDotenv(t, root, `services:
  included:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
`)
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

		var missing *template.MissingRequiredError
		require.ErrorAs(t, err, &missing)
		assert.Equal(t, "TOPO_TEST_GREETING", missing.Variable)
		assert.Nil(t, got)
	})

	t.Run("resolves a required parameter through an interpolated extends path", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  common:
    image: alpine
    environment:
      GREETING: ${TOPO_TEST_GREETING:?Greeting is required}
`)
		t.Setenv("TOPO_TEST_BASE_FILE", base)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends:
      file: ${TOPO_TEST_BASE_FILE}
      service: common
x-topo:
  parameters:
    TOPO_TEST_GREETING: {}
`)
		changes := parameter.Changes{"TOPO_TEST_GREETING": new("hello")}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("rejects a missing required parameter through an interpolated extends path", func(t *testing.T) {
		t.Setenv("TOPO_TEST_GREETING", "")
		base := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  common:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_GREETING:?Greeting is required}"}
`)
		t.Setenv("TOPO_TEST_BASE_FILE", base)
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `services:
  app:
    extends:
      file: ${TOPO_TEST_BASE_FILE}
      service: common
x-topo:
  parameters: {TOPO_TEST_GREETING: {}}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		var missing *template.MissingRequiredError
		require.ErrorAs(t, err, &missing)
		assert.Equal(t, "TOPO_TEST_GREETING", missing.Variable)
		assert.Nil(t, got)
	})
}

func TestConfigureLiteralExtends(t *testing.T) {
	writeIncludedCommonServiceWithDotenv := func(t *testing.T, root string) {
		t.Helper()
		includedDirectory := filepath.Join(root, "included")
		testutil.RequireMkdirAll(t, includedDirectory)
		testutil.RequireWriteComposeFile(t, includedDirectory, `services:
  common:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_EXTENDS_GREETING:?Greeting is required}"}
`)
		testutil.RequireWriteFile(t, filepath.Join(includedDirectory, ".env"), "TOPO_TEST_EXTENDS_GREETING=hello\n")
	}

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

	t.Run("extends a service defined in an included file", func(t *testing.T) {
		testutil.Unsetenv(t, "GREETING")
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, "common.yaml"), "services:\n  common:\n    image: alpine\n")
		path := testutil.RequireWriteComposeFile(t, root, `include: [./common.yaml]
services:
  app:
    extends: common
    environment:
      GREETING: ${GREETING}
x-topo:
  parameters:
    GREETING: {}
`)
		cliResolver, err := parameter.NewCLIResolver([]string{"GREETING=hello"}, nil)
		require.NoError(t, err)
		resolver := parameter.NewStrictResolverChain(cliResolver)

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, parameter.Changes{"GREETING": new("hello")}, got)
	})

	t.Run("resolves an included service's extends file relative to its own directory", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_EXTENDS_GREETING")
		root := t.TempDir()
		shared := filepath.Join(root, "shared")
		testutil.RequireMkdirAll(t, shared)
		testutil.RequireWriteFile(t, filepath.Join(shared, "base.yaml"), `services:
  base:
    image: alpine
    environment: {GREETING: "${TOPO_TEST_EXTENDS_GREETING:?Greeting is required}"}
`)
		testutil.RequireWriteComposeFile(t, shared, `services:
  common:
    extends: {file: ./base.yaml, service: base}
`)
		path := testutil.RequireWriteComposeFile(t, root, `include: [./shared/compose.yaml]
services:
  app:
    extends: common
x-topo:
  parameters: {TOPO_TEST_EXTENDS_GREETING: {}}
`)
		changes := parameter.Changes{"TOPO_TEST_EXTENDS_GREETING": new("hello")}
		resolver := parameter.NewStrictResolverChain(parameter.NewStaticResolver(changes))

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Equal(t, changes, got)
	})

	t.Run("checks an overriding root expression against the root environment", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_EXTENDS_GREETING")
		root := t.TempDir()
		writeIncludedCommonServiceWithDotenv(t, root)
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
services:
  app:
    extends: common
    environment: {GREETING: "${TOPO_TEST_EXTENDS_GREETING:?Greeting is required}"}
x-topo:
  parameters: {TOPO_TEST_EXTENDS_GREETING: {}}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.ErrorContains(t, err, "required variable TOPO_TEST_EXTENDS_GREETING")
		assert.Nil(t, got)
	})

	t.Run("uses the included dotenv with extends service mapping", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_EXTENDS_GREETING")
		root := t.TempDir()
		writeIncludedCommonServiceWithDotenv(t, root)
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
services:
  app:
    extends:
      service: common
x-topo:
  parameters: {TOPO_TEST_EXTENDS_GREETING: {}}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("uses the included dotenv for an inherited required parameter", func(t *testing.T) {
		testutil.Unsetenv(t, "TOPO_TEST_EXTENDS_GREETING")
		root := t.TempDir()
		writeIncludedCommonServiceWithDotenv(t, root)
		path := testutil.RequireWriteComposeFile(t, root, `include: [./included/compose.yaml]
services:
  app:
    extends: common
x-topo:
  parameters: {TOPO_TEST_EXTENDS_GREETING: {}}
`)
		resolver := parameter.NewStrictResolverChain()

		got, err := project.Configure(project.Scope{ComposeFile: path}, resolver)

		require.NoError(t, err)
		assert.Nil(t, got)
	})
}
