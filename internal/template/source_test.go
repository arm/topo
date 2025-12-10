package template_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSource(t *testing.T) {
	t.Run("template source", func(t *testing.T) {
		got, err := template.NewSource("template:hello")

		require.NoError(t, err)
		want := template.TemplateIdSource("hello")
		assert.Equal(t, want, got)
	})

	t.Run("dir source", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  template.DirSource
		}{
			{
				name:  "absolute path",
				input: "dir:/path/to/template",
				want:  template.DirSource{Path: "/path/to/template"},
			},
			{
				name:  "relative path",
				input: "dir:./local/template",
				want:  template.DirSource{Path: "./local/template"},
			},
			{
				name:  "path with spaces",
				input: "dir:/path/with spaces/template",
				want:  template.DirSource{Path: "/path/with spaces/template"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := template.NewSource(tt.input)

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("git source", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  template.GitSource
		}{
			{
				name:  "HTTPS without ref",
				input: "git:https://github.com/user/repo.git",
				want: template.GitSource{
					URL: "https://github.com/user/repo.git",
					Ref: "",
				},
			},
			{
				name:  "HTTPS with # ref",
				input: "git:https://github.com/user/repo.git#develop",
				want: template.GitSource{
					URL: "https://github.com/user/repo.git",
					Ref: "develop",
				},
			},
			{
				name:  "SSH without ref",
				input: "git:git@github.com:user/repo.git",
				want: template.GitSource{
					URL: "git@github.com:user/repo.git",
					Ref: "",
				},
			},
			{
				name:  "SSH with # ref",
				input: "git:git@github.com:user/repo.git#main",
				want: template.GitSource{
					URL: "git@github.com:user/repo.git",
					Ref: "main",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := template.NewSource(tt.input)

				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("error cases", func(t *testing.T) {
		tests := []struct {
			name          string
			input         string
			errorContains string
		}{
			{
				name:          "missing colon",
				input:         "template-ubuntu",
				errorContains: "invalid source format",
			},
			{
				name:          "empty value",
				input:         "template:",
				errorContains: "source value cannot be empty",
			},
			{
				name:          "empty source",
				input:         "",
				errorContains: "invalid source format",
			},
			{
				name:          "unsupported source type",
				input:         "foo:value",
				errorContains: "unsupported source type: foo",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := template.NewSource(tt.input)

				assert.ErrorContains(t, err, tt.errorContains)
			})
		}
	})
}

func TestTemplateIdSource(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		t.Run("returns template ID in correct format", func(t *testing.T) {
			src := template.TemplateIdSource("hello-world")
			assert.Equal(t, "template:hello-world", src.String())
		})
	})
}

func TestGitSource(t *testing.T) {
	t.Run("CopyTo", func(t *testing.T) {
		dstDir := t.TempDir()
		src := template.GitSource{URL: "https://github.com/example/repo.git"}

		err := src.CopyTo(dstDir)

		assert.ErrorIs(t, err, template.DestDirExistsError{Dir: dstDir})
	})

	t.Run("String", func(t *testing.T) {
		t.Run("returns git:URL#ref for HTTPS URLs when ref is set", func(t *testing.T) {
			src := template.GitSource{
				URL: "https://github.com/example/test.git",
				Ref: "v1.0",
			}
			assert.Equal(t, "git:https://github.com/example/test.git#v1.0", src.String())
		})

		t.Run("returns git:URL#ref for SSH URLs when ref is set", func(t *testing.T) {
			src := template.GitSource{
				URL: "git@github.com:example/test.git",
				Ref: "main",
			}
			assert.Equal(t, "git:git@github.com:example/test.git#main", src.String())
		})

		t.Run("returns git:URL when ref is empty", func(t *testing.T) {
			src := template.GitSource{
				URL: "https://github.com/example/test.git",
				Ref: "",
			}
			assert.Equal(t, "git:https://github.com/example/test.git", src.String())
		})
	})
}

func TestDirSource(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		t.Run("returns dir:path format", func(t *testing.T) {
			src := template.DirSource{Path: "/path/to/template"}

			assert.Equal(t, "dir:/path/to/template", src.String())
		})

		t.Run("returns dir:path for relative paths", func(t *testing.T) {
			src := template.DirSource{Path: "./local/template"}

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
			src := template.DirSource{Path: srcDir}

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
			src := template.DirSource{Path: srcDir}

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
			src := template.DirSource{Path: srcDir}

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
			src := template.DirSource{Path: "/nonexistent/path"}
			dstDir := filepath.Join(t.TempDir(), "dest")

			err := src.CopyTo(dstDir)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to access source directory")
		})

		t.Run("errors when source is a file", func(t *testing.T) {
			srcFile := filepath.Join(t.TempDir(), "file.txt")
			require.NoError(t, os.WriteFile(srcFile, []byte("content"), 0o644))
			src := template.DirSource{Path: srcFile}
			dstDir := filepath.Join(t.TempDir(), "dest")

			err := src.CopyTo(dstDir)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "source path is not a directory")
		})

		t.Run("errors when destination is inside source", func(t *testing.T) {
			srcDir := t.TempDir()
			dstDir := filepath.Join(srcDir, "subdir")
			src := template.DirSource{Path: srcDir}

			err := src.CopyTo(dstDir)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "destination directory")
			assert.Contains(t, err.Error(), "is inside source directory")
		})

		t.Run("errors when destination already exists", func(t *testing.T) {
			srcDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0o644))
			dstDir := t.TempDir()
			src := template.DirSource{Path: srcDir}

			err := src.CopyTo(dstDir)

			assert.ErrorIs(t, err, template.DestDirExistsError{Dir: dstDir})
		})
	})
}
