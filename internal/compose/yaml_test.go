package compose_test

import (
	"bytes"
	"testing"

	"github.com/arm-debug/topo-cli/internal/compose"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadNodes(t *testing.T) {
	t.Run("returns error when project file not found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := compose.ReadNodes(dir)

		assert.Error(t, err)
	})

	t.Run("returns a yaml node when project file found", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
name: test
services:
  test-service:
    build:
      context: .
      args:
        FOO: new-foo
        BAR: new-bar
`
		composeFilePath := testutil.WriteComposeFile(t, dir, composeFileContents)
		got, err := compose.ReadNodes(composeFilePath)
		require.NoError(t, err)

		want, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		assert.Equal(t, want, got)
	})
}

func TestApplyArgs(t *testing.T) {
	t.Run("updates all matching services when arg matches in multiple services", func(t *testing.T) {
		composeFileContents := `
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
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)
		args := map[string]string{"FOO": "baz"}

		err = compose.ApplyArgs(proj, args, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(proj)
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
		composeFileContents := `
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
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)
		args := map[string]string{"FOO": "baz"}

		err = compose.ApplyArgs(proj, args, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(proj)
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
		composeFileContents := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: bar
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)
		orig, err := yaml.Marshal(proj)
		require.NoError(t, err)

		err = compose.ApplyArgs(proj, nil, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(proj)
		require.NoError(t, err)
		assert.YAMLEq(t, string(orig), string(got))
	})

	t.Run("when multiple args are provided applies all of them", func(t *testing.T) {
		composeFileContents := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: foo
        BAR: bar
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		args := map[string]string{
			"FOO": "new-foo",
			"BAR": "new-bar",
		}

		err = compose.ApplyArgs(proj, args, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(proj)
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
		composeFileContents := `
services:
  test-service:
    build:
      context: .
      args:
        FOO: foo
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)
		args := map[string]string{"BAR": "baz"}
		buf := &bytes.Buffer{}

		err = compose.ApplyArgs(proj, args, buf)

		require.NoError(t, err)
		assert.Equal(t, "warning: arg \"BAR\" was resolved but not found in any service build args\n", buf.String())
	})

	t.Run("when build args are a YAML sequence applies all resolved values", func(t *testing.T) {
		composeFileContents := `
services:
  test-service:
    build:
      context: .
      args: ["FOO=foo", "BAR"]
`

		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)
		args := map[string]string{
			"FOO": "new-foo",
			"BAR": "new-bar",
		}

		err = compose.ApplyArgs(proj, args, nil)

		require.NoError(t, err)
		got, err := yaml.Marshal(proj)
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
