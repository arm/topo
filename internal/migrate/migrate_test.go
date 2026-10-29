package migrate_test

import (
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/migrate"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToEnv(t *testing.T) {
	t.Run("migrates repeated identical values and replaces only declared parameters", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: current
        OTHER: unchanged
  second:
    build:
      args: ["FOO=current"]
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := `services:
  app:
    build:
      args:
        FOO: ${FOO?configured via topo}
        OTHER: unchanged
  second:
    build:
      args: ["FOO=${FOO?configured via topo}"]
x-topo:
  parameters:
    FOO: {}
`

		err := migrate.ToEnv(path)

		require.NoError(t, err)
		envContents := testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename))
		assert.Equal(t, "FOO=\"current\"\n", envContents)
		assert.NotContains(t, envContents, "OTHER=")
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves build arg interpolation in the env file", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: ${FOO}
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := `services:
  app:
    build:
      args:
        FOO: ${FOO?configured via topo}
x-topo:
  parameters:
    FOO: {}
`

		err := migrate.ToEnv(path)

		require.NoError(t, err)
		assert.Equal(t, "FOO=\"${FOO}\"\n", testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename)))
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves files when there are no parameter values to migrate", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: current
x-topo:
  parameters:
    MISSING: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)

		err := migrate.ToEnv(path)

		require.EqualError(t, err, "no parameter values to migrate; only projects with referenced parameters can be migrated")
		assert.NoFileExists(t, filepath.Join(root, env.DefaultFilename))
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
	})

	t.Run("preserves both files when env already exists", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        FOO: current
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		envPath := filepath.Join(root, env.DefaultFilename)
		testutil.RequireWriteFile(t, envPath, "FOO=existing\n")

		err := migrate.ToEnv(path)

		require.ErrorContains(t, err, "env file already exists")
		assert.Equal(t, "FOO=existing\n", testutil.RequireReadFile(t, envPath))
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
	})

	t.Run("skips declared parameters with no matching build arg", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  app:
    build:
      args:
        PRESENT: ""
x-topo:
  parameters:
    MISSING: {}
    PRESENT: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)
		want := `services:
  app:
    build:
      args:
        PRESENT: ${PRESENT?configured via topo}
x-topo:
  parameters:
    MISSING: {}
    PRESENT: {}
`

		err := migrate.ToEnv(path)

		require.NoError(t, err)
		envContents := testutil.RequireReadFile(t, filepath.Join(root, env.DefaultFilename))
		assert.NotContains(t, envContents, "MISSING=")
		assert.Equal(t, "PRESENT=\"\"\n", envContents)
		assert.YAMLEq(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("rejects parameters with multiple current values", func(t *testing.T) {
		root := t.TempDir()
		contents := `services:
  first:
    build:
      args:
        FOO: current
  second:
    build:
      args:
        FOO: another
x-topo:
  parameters:
    FOO: {}
`
		path := testutil.RequireWriteComposeFile(t, root, contents)

		err := migrate.ToEnv(path)

		require.ErrorContains(t, err, "parameter FOO has more than one current value")
		assert.NoFileExists(t, filepath.Join(root, env.DefaultFilename))
		assert.Equal(t, contents, testutil.RequireReadFile(t, path))
	})
}
