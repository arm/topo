package compose_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/compose"
	"github.com/arm-debug/topo-cli/internal/output/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadNode(t *testing.T) {
	t.Run("parses compose yaml into nodes", func(t *testing.T) {
		composeFileContents := `name: test
services:
  test-service:
    build:
      context: .
`
		composeFileReader := strings.NewReader(composeFileContents)

		got, err := compose.ReadNode(composeFileReader)

		require.NoError(t, err)
		gotYAML, err := yaml.Marshal(got)
		require.NoError(t, err)
		assert.YAMLEq(t, composeFileContents, string(gotYAML))
	})

	t.Run("returns error when compose file is empty", func(t *testing.T) {
		composeFileReader := strings.NewReader("")

		got, err := compose.ReadNode(composeFileReader)

		assert.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("returns error when yaml is invalid", func(t *testing.T) {
		composeFileReader := strings.NewReader("invalid: yaml: content:")

		got, err := compose.ReadNode(composeFileReader)

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

		_, err := compose.ApplyArgs(project, args)

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

		_, err := compose.ApplyArgs(project, args)

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

		_, err := compose.ApplyArgs(project, nil)

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

		_, err := compose.ApplyArgs(project, args)

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

		logs, err := compose.ApplyArgs(project, args)

		require.NoError(t, err)
		assert.Contains(t, logs, logger.Entry{
			Level:   logger.Warning,
			Message: "arg \"BAR\" was resolved but not found in any service build args",
		})
	})

	t.Run("when service has extends and no build injects build args", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  zephyr:
    extends:
      file: zephyr-application/compose.yaml
      service: zephyr
`)
		args := map[string]string{
			"PLATFORM":   "stm32mp257",
			"REMOTEPROC": "m33",
		}

		err := compose.ApplyArgs(project, args, nil)

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
        REMOTEPROC: m33
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("when service has extends and build but no args injects args", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  zephyr:
    extends:
      file: zephyr-application/compose.yaml
      service: zephyr
    build:
      context: .
`)
		args := map[string]string{"PLATFORM": "stm32mp257"}

		err := compose.ApplyArgs(project, args, nil)

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
      context: .
      args:
        PLATFORM: stm32mp257
`
		assert.YAMLEq(t, want, string(got))
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
		args := map[string]string{"PLATFORM": "stm32mp257"}

		err := compose.ApplyArgs(project, args, nil)

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

	t.Run("mixed services with extends and inline build args both get args applied", func(t *testing.T) {
		project := yamlToNode(t, `
services:
  zephyr:
    extends:
      file: zephyr-application/compose.yaml
      service: zephyr
  inline-svc:
    build:
      context: .
      args:
        PLATFORM: old-value
`)
		args := map[string]string{"PLATFORM": "stm32mp257"}

		err := compose.ApplyArgs(project, args, nil)

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
  inline-svc:
    build:
      context: .
      args:
        PLATFORM: stm32mp257
`
		assert.YAMLEq(t, want, string(got))
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

		_, err := compose.ApplyArgs(project, args)

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

func TestRemoveService(t *testing.T) {
	t.Run("errors when specified service is not defined in the yaml", func(t *testing.T) {
		content := `services:
  keepMe:
    build:
      context: ./keepMe
`
		project := yamlToNode(t, content)

		err := compose.RemoveService(project, "removeMe")

		require.ErrorContains(t, err, "service removeMe not found")
	})

	t.Run("removes specified service from compose file", func(t *testing.T) {
		content := `services:
  keepMe:
    build:
      context: ./keepMe
  removeMe:
    build:
      context: ./removeMe
`
		project := yamlToNode(t, content)

		err := compose.RemoveService(project, "removeMe")

		require.NoError(t, err)
		got, err := yaml.Marshal(project)
		require.NoError(t, err)
		want := `
services:
  keepMe:
    build:
      context: ./keepMe
`
		assert.YAMLEq(t, want, string(got))
	})
}

func TestWriteNode(t *testing.T) {
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

		err := compose.WriteNode(project, &buf)
		require.NoError(t, err)

		got := buf.String()
		assert.YAMLEq(t, want, got)
	})
}

func yamlToNode(t *testing.T, yamlContents string) *yaml.Node {
	t.Helper()
	project, err := compose.ReadNode(strings.NewReader(yamlContents))
	require.NoError(t, err)
	return project
}
