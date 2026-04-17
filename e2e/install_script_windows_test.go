//go:build windows

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runInstallScript(t *testing.T, args ...string) (string, error) {
	t.Helper()

	path, err := filepath.Abs("../scripts/install.ps1")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	localAppDataDir := filepath.Join(tmpDir, "localappdata")
	appDataDir := filepath.Join(tmpDir, "appdata")
	tempDir := filepath.Join(tmpDir, "tmp")

	for _, dir := range []string{homeDir, localAppDataDir, appDataDir, tempDir} {
		err := os.MkdirAll(dir, 0o755)
		require.NoError(t, err)
	}

	cmdArgs := append([]string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", path,
	}, args...)
	cmd := exec.Command("powershell", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"HOME="+homeDir,
		"USERPROFILE="+homeDir,
		"LOCALAPPDATA="+localAppDataDir,
		"APPDATA="+appDataDir,
		"TEMP="+tempDir,
		"TMP="+tempDir,
	)

	out, err := cmd.CombinedOutput()
	return string(out), err
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

	t.Run("can reinstall in place", func(t *testing.T) {
		dir := t.TempDir()
		_, err := runInstallScript(t, "-Path", dir)
		require.NoError(t, err)

		_, err = runInstallScript(t, "-Path", dir)

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(dir, "topo.exe"))
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		out, err := runInstallScript(t, "-Bogus")

		assert.Error(t, err)
		assert.Contains(t, out, "Bogus")
	})
}
