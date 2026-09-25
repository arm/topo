package migrate_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm/topo/internal/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadComposeNode(t *testing.T) {
	t.Run("parses compose yaml into nodes", func(t *testing.T) {
		composeFileContents := `name: test
services:
  test-service:
    build:
      context: .
`
		composeFileReader := strings.NewReader(composeFileContents)

		got, err := migrate.ReadComposeNode(composeFileReader)

		require.NoError(t, err)
		gotYAML, err := yaml.Marshal(got)
		require.NoError(t, err)
		assert.YAMLEq(t, composeFileContents, string(gotYAML))
	})

	t.Run("returns error when compose file is empty", func(t *testing.T) {
		composeFileReader := strings.NewReader("")

		got, err := migrate.ReadComposeNode(composeFileReader)

		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("returns error when yaml is invalid", func(t *testing.T) {
		composeFileReader := strings.NewReader("invalid: yaml: content:")

		got, err := migrate.ReadComposeNode(composeFileReader)

		assert.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestApplyParameterValuesToCompose(t *testing.T) {
	for _, value := range []string{"30", "1.5", "true"} {
		t.Run("replaces typed scalar "+value+" with a string reference", func(t *testing.T) {
			project := yamlToNode(t, `services:
  app:
    build:
      args:
        VALUE: `+value+`
`)
			values := map[string]string{"VALUE": "a string"}

			err := migrate.ApplyParameterValuesToCompose(project, values)

			require.NoError(t, err)
			got, err := yaml.Marshal(project)
			require.NoError(t, err)
			want := `services:
  app:
    build:
      args:
        VALUE: a string
`
			assert.YAMLEq(t, want, string(got))
		})
	}

	t.Run("updates all matching services when a parameter matches", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args:
        FOO: bar
  another-service:
    build:
      context: .
      args:
        FOO: elephant
`)
		values := map[string]string{"FOO": "baz"}

		err := migrate.ApplyParameterValuesToCompose(project, values)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		want := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: baz
  another-service:
    build:
      context: .
      args:
        FOO: baz
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("when some services lack build args only matching services are updated", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  with-arg:
    build:
      context: .
      args:
        FOO: bar
  no-build:
    image: busybox
  with-build-no-args:
    build:
      context: .
`)
		values := map[string]string{"FOO": "baz"}

		err := migrate.ApplyParameterValuesToCompose(project, values)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		want := `
services:
  with-arg:
    build:
      context: .
      args:
        FOO: baz
  no-build:
    image: busybox
  with-build-no-args:
    build:
      context: .
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("when no parameters are provided returns nil and leaves project unchanged", func(t *testing.T) {
		yamlContents := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: bar
`
		project := yamlToNode(t, yamlContents)

		err := migrate.ApplyParameterValuesToCompose(project, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		assert.YAMLEq(t, yamlContents, string(got))
	})

	t.Run("when multiple parameters are provided applies all of them", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args:
        FOO: foo
        BAR: bar
`)
		values := map[string]string{
			"FOO": "new-foo",
			"BAR": "new-bar",
		}

		err := migrate.ApplyParameterValuesToCompose(project, values)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		want := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: new-foo
        BAR: new-bar
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("when provided parameters are unused returns nil", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args:
        FOO: foo
`)
		values := map[string]string{"BAR": "baz"}

		err := migrate.ApplyParameterValuesToCompose(project, values)

		require.NoError(t, err)
	})

	t.Run("when service has extends and existing build args updates in place", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  zephyr:
    extends:
      file: zephyr-application/compose.yaml
      service: zephyr
    build:
      args:
        PLATFORM: old-value
`)
		values := map[string]string{"PLATFORM": "stm32mp257"}

		err := migrate.ApplyParameterValuesToCompose(project, values)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		want := `
services:
  zephyr:
    extends:
      file: zephyr-application/compose.yaml
      service: zephyr
    build:
      args:
        PLATFORM: stm32mp257
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("when build args are a YAML sequence applies all parameters", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args: ["FOO=foo", "BAR"]
`)
		values := map[string]string{
			"FOO": "new-foo",
			"BAR": "new-bar",
		}

		err := migrate.ApplyParameterValuesToCompose(project, values)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		want := `
services:
  test-service:
    build:
      context: .
      args: ["FOO=new-foo", "BAR=new-bar"]
`
		assert.YAMLEq(t, want, string(got))
	})
}

func TestWriteComposeNode(t *testing.T) {
	t.Run("writes YAML node to compose file", func(t *testing.T) {
		want := `
name: test
services:
  test-service:
    build:
      context: .
      args: ["FOO=new-foo", "BAR=new-bar"]
`
		project := yamlToNode(t, want)
		var buf bytes.Buffer

		err := migrate.WriteComposeNode(project, &buf)
		require.NoError(t, err)

		got := buf.String()
		assert.YAMLEq(t, want, got)
	})
}

func yamlToNode(t *testing.T, yamlContents string) *yaml.Node {
	t.Helper()
	project, err := migrate.ReadComposeNode(strings.NewReader(yamlContents))
	require.NoError(t, err)
	return project
}
