package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDir(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		t.Run("returns dir:path format", func(t *testing.T) {
			src := source.Dir{Path: "/path/to/template"}

			assert.Equal(t, "dir:/path/to/template", src.String())
		})

		t.Run("returns dir:path for relative paths", func(t *testing.T) {
			src := source.Dir{Path: "./local/template"}

			assert.Equal(t, "dir:./local/template", src.String())
		})
	})

	t.Run("CopyTo", func(t *testing.T) {
		t.Run("copies directory contents to destination", func(t *testing.T) {
			srcDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0o644))
			require.NoError(t, os.Mkdir(filepath.Join(srcDir, "subdir"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "nested.txt"), []byte("nested"), 0o644))
			dstDir := filepath.Join(t.TempDir(), "dest")
			src := source.Dir{Path: srcDir}

			err := src.CopyTo(dstDir)

			require.NoError(t, err)
			content, err := os.ReadFile(filepath.Join(dstDir, "file.txt"))
			require.NoError(t, err)
			assert.Equal(t, "content", string(content))
			nested, err := os.ReadFile(filepath.Join(dstDir, "subdir", "nested.txt"))
			require.NoError(t, err)
			assert.Equal(t, "nested", string(nested))
		})

		t.Run("preserves file permissions", func(t *testing.T) {
			files := []struct {
				name    string
				content string
				perm    os.FileMode
			}{
				{"executable.sh", "#!/bin/bash\necho hello", 0o755},
				{"readonly.txt", "do not modify", 0o444},
				{"config.json", "{}", 0o600},
			}
			srcDir := t.TempDir()
			for _, f := range files {
				require.NoError(t, os.WriteFile(filepath.Join(srcDir, f.name), []byte(f.content), f.perm))
			}
			dstDir := filepath.Join(t.TempDir(), "dest")
			src := source.Dir{Path: srcDir}

			err := src.CopyTo(dstDir)

			require.NoError(t, err)
			for _, f := range files {
				info, err := os.Stat(filepath.Join(dstDir, f.name))
				require.NoError(t, err)
				assert.Equal(t, f.perm, info.Mode().Perm(), "permission mismatch for %s", f.name)
			}
		})

		t.Run("preserves symlinks as symlinks", func(t *testing.T) {
			srcDir := t.TempDir()
			targetFile := filepath.Join(srcDir, "target.txt")
			require.NoError(t, os.WriteFile(targetFile, []byte("target content"), 0o644))
			symlinkPath := filepath.Join(srcDir, "link.txt")
			require.NoError(t, os.Symlink("target.txt", symlinkPath))
			dstDir := filepath.Join(t.TempDir(), "dest")
			src := source.Dir{Path: srcDir}

			err := src.CopyTo(dstDir)

			require.NoError(t, err)
			dstLink := filepath.Join(dstDir, "link.txt")
			info, err := os.Lstat(dstLink)
			require.NoError(t, err)
			assert.True(t, info.Mode()&os.ModeSymlink != 0, "expected symlink")
			target, err := os.Readlink(dstLink)
			require.NoError(t, err)
			assert.Equal(t, "target.txt", target)
		})

		t.Run("errors when source does not exist", func(t *testing.T) {
			src := source.Dir{Path: "/nonexistent/path"}
			dstDir := filepath.Join(t.TempDir(), "dest")

			err := src.CopyTo(dstDir)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to access source directory")
		})

		t.Run("errors when source is a file", func(t *testing.T) {
			srcFile := filepath.Join(t.TempDir(), "file.txt")
			require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))
			src := source.Dir{Path: srcFile}
			dstDir := filepath.Join(t.TempDir(), "dest")

			err := src.CopyTo(dstDir)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "source path is not a directory")
		})

		t.Run("errors when destination is inside source", func(t *testing.T) {
			srcDir := t.TempDir()
			dstDir := filepath.Join(srcDir, "subdir")
			src := source.Dir{Path: srcDir}

			err := src.CopyTo(dstDir)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "destination directory")
			assert.Contains(t, err.Error(), "is inside source directory")
		})

		t.Run("errors when destination already exists", func(t *testing.T) {
			srcDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0o644))
			dstDir := t.TempDir()
			src := source.Dir{Path: srcDir}

			err := src.CopyTo(dstDir)

			assert.ErrorIs(t, err, source.DestDirExistsError{Dir: dstDir})
		})
	})
}
