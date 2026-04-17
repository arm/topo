//go:build !windows
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scriptPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../scripts/install.sh")
	require.NoError(t, err)
	return path
}

func runInstallScript(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmdArgs := append([]string{scriptPath(t)}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInstallScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping install script e2e tests in short mode")
	}

	t.Run("installs latest version", func(t *testing.T) {
		dir := t.TempDir()
		bin := filepath.Join(dir, "topo")

		out, err := runInstallScript(t, "--path", dir)

		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, "Installed topo")
		assert.FileExists(t, bin)
		info, err := os.Stat(bin)
		require.NoError(t, err)
		assert.NotZero(t, info.Mode()&0o111, "binary should be executable")
	})

	t.Run("installs a specific version", func(t *testing.T) {
		version := "v4.0.0"
		dir := t.TempDir()

		out, err := runInstallScript(t, "--version", version, "--path", dir)

		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, version)
		assert.FileExists(t, filepath.Join(dir, "topo"))
	})

	t.Run("can reinstall in-place", func(t *testing.T) {
		dir := t.TempDir()
		_, err := runInstallScript(t, "--path", dir)
		require.NoError(t, err)

		_, err = runInstallScript(t, "--path", dir)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(dir, "topo"))
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		out, err := runInstallScript(t, "--bogus")

		assert.Error(t, err)
		assert.Contains(t, out, "Unknown option")
	})
}
