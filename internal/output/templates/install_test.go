package templates_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/install"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallResults(t *testing.T) {
	t.Run("AsJSON", func(t *testing.T) {
		t.Run("returns empty array for no results", func(t *testing.T) {
			results := templates.InstallResults{}

			var out bytes.Buffer

			err := printable.Print(
				results,
				&out,
				term.JSON,
			)
			require.NoError(t, err)

			assert.JSONEq(t, `[]`, out.String())
		})

		t.Run("returns JSON array with install locations", func(t *testing.T) {
			results := templates.InstallResults{
				{
					Location: install.PathCandidate{Path: "/usr/local/bin", OnPath: true},
					Binary:   "foo",
				},
				{
					Location: install.PathCandidate{Path: "/usr/bin", OnPath: false},
					Binary:   "bar",
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				results,
				&out,
				term.JSON,
			)
			require.NoError(t, err)

			want := `[
				{"path":"/usr/local/bin","on_path":true,"binary":"foo"},
				{"path":"/usr/bin","on_path":false,"binary":"bar"}
			]`

			assert.JSONEq(t, want, out.String())
		})
	})

	t.Run("AsPlain", func(t *testing.T) {
		t.Run("returns message for no results", func(t *testing.T) {
			results := templates.InstallResults{}

			var out bytes.Buffer

			err := printable.Print(
				results,
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			assert.Equal(t, "No binaries installed", out.String())
		})

		t.Run("returns success message for single binary on PATH", func(t *testing.T) {
			results := templates.InstallResults{
				{
					Location: install.PathCandidate{Path: "/usr/local/bin", OnPath: true},
					Binary:   "my-binary",
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				results,
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			got := out.String()
			assert.Contains(t, got, "Installed my-binary to /usr/local/bin")
			assert.NotContains(t, got, "not on your PATH")
		})

		t.Run("includes PATH warning when installed to directory not on PATH", func(t *testing.T) {
			results := templates.InstallResults{
				{
					Location: install.PathCandidate{Path: "~/bin", OnPath: false},
					Binary:   "my-binary",
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				results,
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			got := out.String()
			assert.Contains(t, got, "Installed my-binary to ~/bin")
			assert.Contains(t, got, "~/bin is not on your PATH")
			assert.Contains(t, got, `export PATH="$PATH:~/bin"`)
		})

		t.Run("groups multiple binaries in same off-PATH directory", func(t *testing.T) {
			results := templates.InstallResults{
				{
					Location: install.PathCandidate{Path: "~/bin", OnPath: false},
					Binary:   "foo",
				},
				{
					Location: install.PathCandidate{Path: "~/bin", OnPath: false},
					Binary:   "bar",
				},
			}

			var out bytes.Buffer

			err := printable.Print(
				results,
				&out,
				term.Plain,
			)
			require.NoError(t, err)

			got := out.String()
			assert.Contains(t, got, "foo")
			assert.Contains(t, got, "bar")
			assert.Contains(t, got, "~/bin is not on your PATH")
		})
	})
}
