package compose_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadNodes(t *testing.T) {
	t.Run("parses compose yaml into nodes", func(t *testing.T) {
		composeFileContents := `name: test
services:
  test-service:
    build:
      context: .
`
		composeFileReader := strings.NewReader(composeFileContents)

		got, err := compose.ReadNodes(composeFileReader)

		require.NoError(t, err)
		gotYAML, err := yaml.Marshal(got)
		require.NoError(t, err)
		assert.YAMLEq(t, composeFileContents, string(gotYAML))
	})

	t.Run("returns error when compose file is empty", func(t *testing.T) {
		composeFileReader := strings.NewReader("")

		got, err := compose.ReadNodes(composeFileReader)

		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("returns error when yaml is invalid", func(t *testing.T) {
		composeFileReader := strings.NewReader("invalid: yaml: content:")

		got, err := compose.ReadNodes(composeFileReader)

		assert.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestApplyArgs(t *testing.T) {
	t.Run("updates all matching services when arg matches in multiple services", func(t *testing.T) {
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
		args := map[string]string{"FOO": "baz"}

		err := compose.ApplyArgs(project, args, nil)

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

	t.Run("when some services lack args only matching services are updated", func(t *testing.T) {
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
		args := map[string]string{"FOO": "baz"}

		err := compose.ApplyArgs(project, args, nil)

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

	t.Run("when no args are provided returns nil and leaves project unchanged ", func(t *testing.T) {
		yamlContents := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: bar
`
		project := yamlToNode(t, yamlContents)

		err := compose.ApplyArgs(project, nil, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		assert.YAMLEq(t, yamlContents, string(got))
	})

	t.Run("when multiple args are provided applies all of them", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args:
        FOO: foo
        BAR: bar
`)
		args := map[string]string{
			"FOO": "new-foo",
			"BAR": "new-bar",
		}

		err := compose.ApplyArgs(project, args, nil)

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

	t.Run("when resolved args are unused writes warning to provided writer", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args:
        FOO: foo
`)
		args := map[string]string{"BAR": "baz"}
		buf := &bytes.Buffer{}

		err := compose.ApplyArgs(project, args, buf)

		require.NoError(t, err)
		assert.Equal(t, "warning: arg \"BAR\" was resolved but not found in any service build args\n", buf.String())
	})

	t.Run("when build args are a YAML sequence applies all resolved values", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  test-service:
    build:
      context: .
      args: ["FOO=foo", "BAR"]
`)
		args := map[string]string{
			"FOO": "new-foo",
			"BAR": "new-bar",
		}

		err := compose.ApplyArgs(project, args, nil)

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

func yamlToNode(t *testing.T, yamlContents string) *yaml.Node {
	t.Helper()
	project, err := compose.ReadNodes(strings.NewReader(yamlContents))
	require.NoError(t, err)
	return project
}
