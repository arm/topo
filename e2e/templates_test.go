package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatesTargetDescription(t *testing.T) {
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
		assert.Contains(t, output, "✅ Hello-World")
		assert.Contains(t, output, "❌ Lightbulb-moment")
	})
	t.Run("correctly handles the --target flag when no target description is provided", func(t *testing.T) {
		bin := buildBinary(t)
		target := testutil.StartTargetContainer(t)

		cmd := exec.Command(bin, "templates", "--target", target.SSHDestination)
		out, err := cmd.CombinedOutput()
		output := string(out)

		require.NoError(t, err, output)
		assert.Contains(t, output, "✅ Hello-World")
	})
}

func writeTargetDescription(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target-description.yaml")

	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}
