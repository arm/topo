package parse_test

import (
	"bytes"
	"testing"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/project/parse"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestListArgs(t *testing.T) {
	t.Run("when present returns an array of args", func(t *testing.T) {
		composeFileContents := `
services:
  ollama-service:
    build:
      context: .
      args:
        FOO: bar

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "a dummy argument"
      required: true
      example: bar
`
		proj, _ := testutil.ParseYAMLString(composeFileContents)

		args := parse.ListArgs(proj)

		want := []arguments.Arg{{
			Name:        "FOO",
			Description: "a dummy argument",
			Required:    true,
			Example:     "bar",
		}}
		assert.Equal(t, want, args)
	})

	t.Run("when none present returns an empty array of args ", func(t *testing.T) {
		composeFileContents := `
services:
  ollama-service:
    build:
      context: .
      args:
        FOO: bar

x-topo:
  name: "My Project"
`
		proj, _ := testutil.ParseYAMLString(composeFileContents)

		args := parse.ListArgs(proj)

		want := []arguments.Arg{}
		assert.Equal(t, want, args)
	})

	t.Run("when x-topo is not present returns an empty list", func(t *testing.T) {
		composeFileContents := `
services:
  ollama-service:
    build:
      context: .
      args:
        FOO: bar
`
		proj, _ := testutil.ParseYAMLString(composeFileContents)

		args := parse.ListArgs(proj)
		assert.Empty(t, args)
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "a dummy argument"
      required: true
      example: bar
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		resolved := []arguments.ResolvedArg{{
			Name:  "FOO",
			Value: "baz",
		}}

		err = parse.ApplyArgs(proj, resolved, nil)
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "a dummy argument"
      required: true
      example: bar
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "a dummy argument"
      required: true
      example: bar
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		resolved := []arguments.ResolvedArg{{
			Name:  "FOO",
			Value: "baz",
		}}

		err = parse.ApplyArgs(proj, resolved, nil)
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "a dummy argument"
      required: true
      example: bar
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

x-topo:
  name: "My Project"
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		orig, err := yaml.Marshal(proj)
		require.NoError(t, err)

		err = parse.ApplyArgs(proj, nil, nil)
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "first arg"
      required: true
      example: foo
    BAR:
      description: "second arg"
      required: false
      example: bar
`
		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		resolved := []arguments.ResolvedArg{
			{Name: "FOO", Value: "new-foo"},
			{Name: "BAR", Value: "new-bar"},
		}

		err = parse.ApplyArgs(proj, resolved, nil)
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "first arg"
      required: true
      example: foo
    BAR:
      description: "second arg"
      required: false
      example: bar
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

		resolved := []arguments.ResolvedArg{
			{Name: "BAR", Value: "baz"},
		}

		buf := &bytes.Buffer{}

		err = parse.ApplyArgs(proj, resolved, buf)
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

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "first arg"
      required: true
      example: foo
    BAR:
      description: "second arg"
      required: false
      example: bar
`

		proj, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		resolved := []arguments.ResolvedArg{
			{Name: "FOO", Value: "new-foo"},
			{Name: "BAR", Value: "new-bar"},
		}

		err = parse.ApplyArgs(proj, resolved, nil)
		require.NoError(t, err)

		got, err := yaml.Marshal(proj)
		require.NoError(t, err)

		want := `
services:
  test-service:
    build:
      context: .
      args: ["FOO=new-foo", "BAR=new-bar"]

x-topo:
  name: "My Project"
  args:
    FOO:
      description: "first arg"
      required: true
      example: foo
    BAR:
      description: "second arg"
      required: false
      example: bar
`
		assert.YAMLEq(t, want, string(got))
	})
}

func TestRead(t *testing.T) {
	t.Run("when project file not found returns error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := parse.Read(dir)

		assert.Error(t, err)
	})

	t.Run("when project file found returns correct project type", func(t *testing.T) {
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
		proj, err := parse.Read(composeFilePath)
		require.NoError(t, err)

		got, err := yaml.Marshal(proj)
		require.NoError(t, err)

		assert.YAMLEq(t, composeFileContents, string(got))
	})
}

func TestReadNodes(t *testing.T) {
	t.Run("returns error when project file not found", func(t *testing.T) {
		dir := t.TempDir()

		_, err := parse.ReadNodes(dir)

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
		got, err := parse.ReadNodes(composeFilePath)
		require.NoError(t, err)

		want, err := testutil.ParseYAMLString(composeFileContents)
		require.NoError(t, err)

		assert.Equal(t, want, got)
	})
}
