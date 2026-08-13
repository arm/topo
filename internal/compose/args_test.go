package compose_test

import (
	"strings"
	"testing"

	"github.com/arm/topo/internal/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, nil)

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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, args)

		require.NoError(t, err)
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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, args)

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

		err := compose.ApplyArgs(project, args)

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
	project, err := compose.ReadNode(strings.NewReader(yamlContents))
	require.NoError(t, err)
	return project
}
