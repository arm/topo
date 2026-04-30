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
	return runInstallScriptWithEnv(t, nil, args...)
}

func runInstallScriptWithEnv(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	cmdArgs := append([]string{scriptPath(t)}, args...)
	cmd := exec.Command("sh", cmdArgs...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func installBinDir(home string) string {
	return filepath.Join(home, ".local", "bin")
}

func TestInstallScript(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping install script e2e tests in short mode")
	}

	t.Run("installs latest version to $HOME/.local/bin", func(t *testing.T) {
		home := t.TempDir()

		out, err := runInstallScriptWithEnv(t, []string{
			"HOME=" + home,
		})

		wantBin := filepath.Join(installBinDir(home), "topo")
		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, "Installed topo")
		assert.FileExists(t, wantBin)
		info, err := os.Stat(wantBin)
		require.NoError(t, err)
		assert.NotZero(t, info.Mode()&0o111, "binary should be executable")
	})

	t.Run("installs a specific version", func(t *testing.T) {
		version := "v4.0.0"
		dir := t.TempDir()

		out, err := runInstallScript(t, "--version", version, "--path", dir)

		wantBin := filepath.Join(dir, "topo")
		require.NoError(t, err, "script failed: %s", out)
		assert.Contains(t, out, version)
		assert.FileExists(t, wantBin)
		// TODO assert that the version of the installed binary matches the requested version
	})

	t.Run("installs to custom path when requested", func(t *testing.T) {
		dir := t.TempDir()

		out, err := runInstallScript(t, "--path", dir)

		require.NoError(t, err, "script failed: %s", out)
		assert.FileExists(t, filepath.Join(dir, "topo"))
	})

	t.Run("can reinstall in-place", func(t *testing.T) {
		home := t.TempDir()
		installDir := installBinDir(home)
		env := []string{
			"HOME=" + home,
		}
		_, err := runInstallScriptWithEnv(t, env, "--version", "v4.0.0")
		require.NoError(t, err)

		_, err = runInstallScriptWithEnv(t, env)

		wantBin := filepath.Join(installDir, "topo")
		require.NoError(t, err)
		assert.FileExists(t, wantBin)
	})

	t.Run("does not prompt to add to PATH when install directory is already on PATH", func(t *testing.T) {
		home := t.TempDir()
		env := []string{
			"HOME=" + home,
			"PATH=" + installBinDir(home) + string(os.PathListSeparator) + os.Getenv("PATH"),
		}
		out, err := runInstallScriptWithEnv(t, env)

		require.NoError(t, err)
		assert.NotContains(t, out, "is not on your PATH")
	})

	t.Run("fails on unknown flag", func(t *testing.T) {
		out, err := runInstallScript(t, "--bogus")

		assert.Error(t, err)
		assert.Contains(t, out, "Unknown option")
	})
}
