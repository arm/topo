package output_test

import (
	"bytes"
	"testing"

	"github.com/arm-debug/topo-cli/internal/install"
	"github.com/arm-debug/topo-cli/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallResults(t *testing.T) {
	t.Run("AsJSON", func(t *testing.T) {
		t.Run("returns empty array for no results", func(t *testing.T) {
			results := output.InstallResults{}

			got, err := results.AsJSON()

			require.NoError(t, err)
			assert.JSONEq(t, `[]`, got)
		})

		t.Run("returns JSON array with install locations", func(t *testing.T) {
			results := output.InstallResults{
				install.InstallResult{
					Location: install.PathCandidate{Path: "/usr/local/bin", OnPath: true},
					Binary:   "foo",
				},
				install.InstallResult{
					Location: install.PathCandidate{Path: "/usr/bin", OnPath: false},
					Binary:   "bar",
				},
			}

			got, err := results.AsJSON()

			require.NoError(t, err)
			want := `[
				{"path":"/usr/local/bin","on_path":true,"binary":"foo"},
				{"path":"/usr/bin","on_path":false,"binary":"bar"}
			]`
			assert.JSONEq(t, want, got)
		})
	})

	t.Run("AsPlain", func(t *testing.T) {
		t.Run("returns message for no results", func(t *testing.T) {
			results := output.InstallResults{}
			initTemplateForTest(t, results)

			got, err := results.AsPlain()

			require.NoError(t, err)
			assert.Equal(t, "No binaries installed", got)
		})

		t.Run("returns success message for single binary on PATH", func(t *testing.T) {
			results := output.InstallResults{
				install.InstallResult{
					Location: install.PathCandidate{Path: "/usr/local/bin", OnPath: true},
					Binary:   "my-binary",
				},
			}
			initTemplateForTest(t, results)

			got, err := results.AsPlain()

			require.NoError(t, err)
			assert.Contains(t, got, "Installed my-binary to /usr/local/bin")
			assert.NotContains(t, got, "not on your PATH")
		})

		t.Run("includes PATH warning when installed to directory not on PATH", func(t *testing.T) {
			results := output.InstallResults{
				install.InstallResult{
					Location: install.PathCandidate{Path: "~/bin", OnPath: false},
					Binary:   "my-binary",
				},
			}
			initTemplateForTest(t, results)

			got, err := results.AsPlain()

			require.NoError(t, err)
			assert.Contains(t, got, "Installed my-binary to ~/bin")
			assert.Contains(t, got, "~/bin is not on your PATH")
			assert.Contains(t, got, "export PATH")
		})

		t.Run("groups multiple binaries in same off-PATH directory", func(t *testing.T) {
			results := output.InstallResults{
				install.InstallResult{
					Location: install.PathCandidate{Path: "~/bin", OnPath: false},
					Binary:   "foo",
				},
				install.InstallResult{
					Location: install.PathCandidate{Path: "~/bin", OnPath: false},
					Binary:   "bar",
				},
			}
			initTemplateForTest(t, results)

			got, err := results.AsPlain()

			require.NoError(t, err)
			// The order of foo, bar may vary due to map iteration
			assert.Contains(t, got, "foo")
			assert.Contains(t, got, "bar")
		})
	})
}

// initTemplateForTest initializes the template by calling PrintInstallResults.
func initTemplateForTest(t *testing.T, results output.InstallResults) {
	t.Helper()
	var buf bytes.Buffer
	printer := output.NewPrinter(&buf, output.PlainFormat)
	_ = output.PrintInstallResults(printer, results)
}
