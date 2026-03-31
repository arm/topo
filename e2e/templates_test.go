package e2e

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplates(t *testing.T) {
	bin := buildBinary(t)

	t.Run("lists builtin templates", func(t *testing.T) {
		cmd := exec.Command(bin, "templates")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		output := string(out)

		assert.Contains(t, output, "topo-welcome")
		assert.Contains(t, output, "https://github.com/")
		assert.Contains(t, output, "Features:")
	})

	t.Run("filtering", func(t *testing.T) {
		t.Run("matches templates to target description", func(t *testing.T) {
			bin := buildBinary(t)

			targetDescriptionYAML := `host:
  - model: Cortex-A
    cores: 4
    features:
      - asimd
totalmemory_kb: 4194304
`
			targetDescriptionPath := writeTargetDescription(t, targetDescriptionYAML)

			cmd := exec.Command(bin, "templates", "--target-description", targetDescriptionPath)
			out, err := cmd.CombinedOutput()
			output := string(out)

			require.NoError(t, err, output)
			assert.Contains(t, output, "✅ topo-welcome")
			assert.Contains(t, output, "❌ topo-lightbulb-moment")
		})

		t.Run("correctly handles the --target flag when no target description is provided", func(t *testing.T) {
			bin := buildBinary(t)
			target := testutil.StartTargetContainer(t)

			cmd := exec.Command(bin, "templates", "--target", target.SSHDestination)
			out, err := cmd.CombinedOutput()
			output := string(out)

			require.NoError(t, err, output)
			assert.Contains(t, output, "✅ topo-welcome")
		})
	})

	t.Run("outputs JSON when specified", func(t *testing.T) {
		cmd := exec.Command(bin, "templates", "--output", "json")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		assert.NoError(t, err)
		var entry templates.RepoCollection
		err = json.Unmarshal(out, &entry)
		assert.NoError(t, err)
	})
}

func writeTargetDescription(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target-description.yaml")

	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
