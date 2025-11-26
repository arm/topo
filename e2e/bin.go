package e2e

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "topo")
	cmd := exec.Command("go", "build", "-o", bin, "../cmd/topo")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "build failed: %s", out)
	return bin
}
