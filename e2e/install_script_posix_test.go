//go:build !windows

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

func scriptPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../scripts/install.sh")
	require.NoError(t, err)
	return path
}

func runInstallScriptWithEnv(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	cmdArgs := append([]string{scriptPath(t)}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runInstallScript(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runInstallScriptWithEnv(t, nil, args...)
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

	t.Run("can override existing binary with --path", func(t *testing.T) {
		dir := t.TempDir()
		_, err := runInstallScript(t, "--path", dir)
		require.NoError(t, err)

		_, err = runInstallScript(t, "--path", dir)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(dir, "topo"))
	})

	t.Run("tells user to use upgrade when topo is already installed", func(t *testing.T) {
		dir := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(dir, "topo"), "")
		pathWithTopo := dir + string(os.PathListSeparator) + os.Getenv("PATH")

		out, err := runInstallScriptWithEnv(t, []string{"PATH=" + pathWithTopo})

		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, "topo is already installed")
		assert.Contains(t, out, "topo upgrade")
		assert.NotContains(t, out, "Installing topo")
		assert.NotContains(t, out, "Installed topo")
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		out, err := runInstallScript(t, "--bogus")

		assert.Error(t, err)
		assert.Contains(t, out, "Unknown option")
	})
}
