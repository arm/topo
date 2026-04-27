//go:build windows

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

func runInstallScriptWithEnv(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()

	path, err := filepath.Abs("../scripts/install.ps1")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	localAppDataDir := filepath.Join(tmpDir, "localappdata")
	testutil.RequireMkdirAll(t, localAppDataDir)

	cmdArgs := append([]string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", path,
	}, args...)
	cmd := exec.Command("powershell", cmdArgs...)
	cmd.Env = append(os.Environ(), env...)

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runInstallScript(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runInstallScriptWithEnv(t, nil, args...)
}

func TestInstallScriptWindows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping install script e2e tests in short mode")
	}

	t.Run("installs latest version", func(t *testing.T) {
		dir := t.TempDir()
		bin := filepath.Join(dir, "topo.exe")

		out, err := runInstallScript(t, "-Path", dir)

		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, "Installed topo")
		assert.FileExists(t, bin)
	})

	t.Run("installs a specific version", func(t *testing.T) {
		version := "v4.0.0"
		dir := t.TempDir()

		out, err := runInstallScript(t, "-Version", version, "-Path", dir)

		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, version)
		assert.FileExists(t, filepath.Join(dir, "topo.exe"))
	})

	t.Run("can override existing binary with -Path", func(t *testing.T) {
		dir := t.TempDir()
		_, err := runInstallScript(t, "-Path", dir)
		require.NoError(t, err)

		_, err = runInstallScript(t, "-Path", dir)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(dir, "topo.exe"))
	})

	t.Run("tells user to use upgrade when topo is already installed", func(t *testing.T) {
		dir := t.TempDir()
		requireWriteDummyExecutable(t, filepath.Join(dir, "topo.exe"))
		pathWithTopo := dir + string(os.PathListSeparator) + os.Getenv("PATH")

		out, err := runInstallScriptWithEnv(t, []string{"PATH=" + pathWithTopo})

		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, "topo is already installed")
		assert.Contains(t, out, "topo upgrade")
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		out, err := runInstallScript(t, "-Bogus")

		assert.Error(t, err)
		assert.Contains(t, out, "Bogus")
	})
}
