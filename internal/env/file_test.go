package env_test

import (
	"fmt"
	"os"
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

func TestUpdateFile(t *testing.T) {
	t.Run("preserves unrelated expressions and assignment formatting", func(t *testing.T) {
		path := writeEnvFile(t, "# greeting\r\nGREETING=${NAME:?required}\r\n export OTHER : old # keep\r\n")

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "# greeting\r\nGREETING=${NAME:?required}\r\n export OTHER : \"updated\" # keep\r\n", testutil.RequireReadFile(t, path))
	})

	t.Run("updates every duplicate assignment", func(t *testing.T) {
		path := writeEnvFile(t, "OTHER=\nCOPY=${OTHER}\nOTHER=last")

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "OTHER=\"updated\"\nCOPY=${OTHER}\nOTHER=\"updated\"", testutil.RequireReadFile(t, path))
	})

	t.Run("replaces a whole multiline value including escaped quotes", func(t *testing.T) {
		path := writeEnvFile(t, `OTHER="one\"two
three" # keep
`)

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "OTHER=\"updated\" # keep\n", testutil.RequireReadFile(t, path))
	})

	t.Run("does not treat multiline contents as assignments", func(t *testing.T) {
		const text = "TEXT='one\nOTHER=not an assignment\nthree'\n"
		path := writeEnvFile(t, text+"OTHER=old\n")

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, text+"OTHER=\"updated\"\n", testutil.RequireReadFile(t, path))
	})

	t.Run("separates appended entries from an unterminated final line", func(t *testing.T) {
		path := writeEnvFile(t, "# comment")

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "# comment\nOTHER=\"updated\"\n", testutil.RequireReadFile(t, path))
	})

	t.Run("can preserve interpolation in replaced and appended values", func(t *testing.T) {
		path := writeEnvFile(t, "NAME=World\nGREETING=old\n")
		updates := map[string]string{"GREETING": "Hello ${NAME}", "FAREWELL": "Bye ${NAME}"}

		err := env.UpdateFile(path, updates, env.EncodeOptions{PreserveInterpolation: true})

		require.NoError(t, err)
		assert.Equal(t, "NAME=World\nGREETING=\"Hello ${NAME}\"\nFAREWELL=\"Bye ${NAME}\"\n", testutil.RequireReadFile(t, path))
	})

	t.Run("creates missing files with sorted entries", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")

		err := env.UpdateFile(path, map[string]string{"B": "two", "A": "one"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "A=\"one\"\nB=\"two\"\n", testutil.RequireReadFile(t, path))
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("escapes interpolation characters, preserving their literal values", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".env")
		values := map[string]string{"VALUE": "${MISSING:?required} $NAME $$ \\ \" ' \n\r\t #"}

		err := env.UpdateFile(path, values, env.EncodeOptions{})

		require.NoError(t, err)
		testutil.RequireEnvFileValues(t, path, values)
	})

	t.Run("leaves the file untouched when a quote is unterminated", func(t *testing.T) {
		const content = "OTHER=old\nBROKEN='unterminated\n"
		path := writeEnvFile(t, content)

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.ErrorContains(t, err, "unterminated")
		assert.Equal(t, content, testutil.RequireReadFile(t, path))
	})

	t.Run("leaves the file untouched when assignment syntax is unsupported", func(t *testing.T) {
		const content = "OTHER=old\nINHERITED\n"
		path := writeEnvFile(t, content)

		err := env.UpdateFile(path, map[string]string{"OTHER": "updated"}, env.EncodeOptions{})

		require.ErrorContains(t, err, "unsupported")
		assert.Equal(t, content, testutil.RequireReadFile(t, path))
	})

	t.Run("returns read errors", func(t *testing.T) {
		err := env.UpdateFile(t.TempDir(), map[string]string{"A": "value"}, env.EncodeOptions{})

		require.ErrorContains(t, err, "failed to read env file")
	})

	t.Run("returns write errors", func(t *testing.T) {
		err := env.UpdateFile(filepath.Join(t.TempDir(), "missing", ".env"), map[string]string{"A": "value"}, env.EncodeOptions{})

		require.ErrorContains(t, err, "failed to write env file")
	})
}

func TestToString(t *testing.T) {
	t.Run("returns a string with sorted entries", func(t *testing.T) {
		content, err := env.ToString(map[string]string{"B": "two", "A": "one"}, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "A=\"one\"\nB=\"two\"\n", content)
	})

	t.Run("escapes special characters in literal values", func(t *testing.T) {
		values := map[string]string{"VALUE": "${MISSING:?required} $NAME $$ \\ \" ' \n\r\t #"}

		content, err := env.ToString(values, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, `VALUE="$${MISSING:?required} $$NAME $$$$ \\ \" ' \n\r\t #"`+"\n", content)
	})

	t.Run("preserves interpolation when requested", func(t *testing.T) {
		values := map[string]string{"GREETING": "Hello ${NAME}"}

		content, err := env.ToString(values, env.EncodeOptions{PreserveInterpolation: true})

		require.NoError(t, err)
		assert.Equal(t, "GREETING=\"Hello ${NAME}\"\n", content)
	})

	t.Run("returns an empty string for no values", func(t *testing.T) {
		content, err := env.ToString(nil, env.EncodeOptions{})

		require.NoError(t, err)
		assert.Equal(t, "", content)
	})
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), env.DefaultFilename)
	testutil.RequireWriteFile(t, path, content)
	return path
}
