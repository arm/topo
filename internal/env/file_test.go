package env_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFiles(t *testing.T) {
	t.Run("resolves files relative to root", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, ".env.first"), "")
		testutil.RequireWriteFile(t, filepath.Join(root, ".env.second"), "")

		paths, err := env.ResolveFiles(root, []string{".env.second", ".env.first"}, false)

		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, ".env.second"), filepath.Join(root, ".env.first")}, paths)
	})

	t.Run("preserves absolute paths alongside relative paths in order", func(t *testing.T) {
		root := t.TempDir()
		firstPath := filepath.Join(t.TempDir(), ".env.external")
		relativeFilename := ".env"
		secondPath := filepath.Join(root, relativeFilename)
		testutil.RequireWriteFile(t, firstPath, "")
		testutil.RequireWriteFile(t, secondPath, "")

		paths, err := env.ResolveFiles(root, []string{firstPath, relativeFilename}, false)

		require.NoError(t, err)
		assert.Equal(t, []string{firstPath, secondPath}, paths)
	})

	t.Run("skips missing optional files", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, ".env"), "")

		paths, err := env.ResolveFiles(root, []string{"missing", ".env", "also-missing"}, true)

		require.NoError(t, err)
		assert.Equal(t, []string{filepath.Join(root, ".env")}, paths)
	})

	t.Run("rejects missing required files without partial results", func(t *testing.T) {
		root := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(root, ".env"), "")

		paths, err := env.ResolveFiles(root, []string{".env", "missing"}, false)

		assert.EqualError(t, err, fmt.Sprintf("env file %q does not exist", filepath.Join(root, "missing")))
		assert.Nil(t, paths)
	})

	t.Run("rejects invalid paths even when missing files are allowed", func(t *testing.T) {
		root := t.TempDir()

		paths, err := env.ResolveFiles(root, []string{"invalid\x00.env"}, true)

		require.ErrorContains(t, err, "failed to check env file")
		assert.Nil(t, paths)
	})
}

func TestReadFiles(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "returns values from an env file",
			content: "GREETING=Hello\nPORT=8080\n",
			want:    map[string]string{"GREETING": "Hello", "PORT": "8080"},
		},
		{
			name:    "preserves explicitly empty values",
			content: "GREETING=\n",
			want:    map[string]string{"GREETING": ""},
		},
		{
			name: "returns an empty map for an empty file",
			want: map[string]string{},
		},
		{
			name:    "preserves single quoted values literally",
			content: "GREETING='Hello # ${NAME}'\n",
			want:    map[string]string{"GREETING": "Hello # ${NAME}"},
		},
		{
			name:    "resolves references to values in the file",
			content: "NAME=World\nGREETING=Hello ${NAME}\n",
			want:    map[string]string{"NAME": "World", "GREETING": "Hello World"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".env")
			testutil.RequireWriteFile(t, path, test.content)

			got, err := env.ReadFiles([]string{path})

			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}

	t.Run("later files override values and reference earlier files", func(t *testing.T) {
		root := t.TempDir()
		first, second := filepath.Join(root, "first.env"), filepath.Join(root, "second.env")
		testutil.RequireWriteFile(t, first, "TOPO_TEST_NAME=World\nGREETING=old\n")
		testutil.RequireWriteFile(t, second, "GREETING=Hello ${TOPO_TEST_NAME}\n")

		got, err := env.ReadFiles([]string{first, second})

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"TOPO_TEST_NAME": "World", "GREETING": "Hello World"}, got)
	})

	t.Run("returns an error when the file does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")

		got, err := env.ReadFiles([]string{path})

		assert.ErrorContains(t, err, "couldn't find env file")
		assert.Nil(t, got)
	})

	t.Run("returns an error when the path is a directory", func(t *testing.T) {
		path := t.TempDir()

		got, err := env.ReadFiles([]string{path})

		assert.ErrorContains(t, err, "failed to read env files")
		assert.Nil(t, got)
	})

	t.Run("does not return partial values when parsing fails", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")
		testutil.RequireWriteFile(t, path, `NAME=World
GREETING="unterminated
`)

		got, err := env.ReadFiles([]string{path})

		assert.ErrorContains(t, err, "failed to read env files")
		assert.Nil(t, got)
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("rewrites the file with a generated header and sorted parameters", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), env.DefaultFilename)
		testutil.RequireWriteFile(t, path, "OLD=value\n")
		values := map[string]string{"PORT": "8080", "GREETING": "Hello"}

		err := env.WriteFile(path, values)

		require.NoError(t, err)
		want := `# Generated by Topo (https://github.com/arm/topo). Do not edit manually.
# Use 'topo configure' to update project parameters.

GREETING="Hello"
PORT="8080"
`
		assert.Equal(t, want, testutil.RequireReadFile(t, path))
	})

	t.Run("rejects empty parameter names", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), env.DefaultFilename)

		err := env.WriteFile(path, map[string]string{"": "Hello"})

		assert.ErrorContains(t, err, "env parameter name must not be empty")
	})

	t.Run("returns an error when the destination is a directory", func(t *testing.T) {
		path := t.TempDir()

		err := env.WriteFile(path, map[string]string{"GREETING": "Hello"})

		assert.ErrorContains(t, err, "failed to write env file")
	})
}
