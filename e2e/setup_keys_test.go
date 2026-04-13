package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupKeys(t *testing.T) {
	target := testutil.StartSSHContainer(t)
	topo := buildBinary(t)

	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")

	out, err := runSetupKeys(topo, target.SSHDestination, home)
	require.NoError(t, err, "setup-keys failed: %s", out)

	privKeyPath := findGeneratedKey(t, sshDir)
	assert.FileExists(t, privKeyPath)
	assert.FileExists(t, privKeyPath+".pub")

	authorizedKeys := readRemoteFile(t, target, "/root/.ssh/authorized_keys")
	pubKey := testutil.RequireReadFile(t, privKeyPath+".pub")
	assert.Contains(t, authorizedKeys, strings.TrimSpace(string(pubKey)))

	sshOut, err := sshWithKey(target.SSHDestination, privKeyPath, "echo", "authenticated")
	require.NoError(t, err, "ssh with key failed: %s", sshOut)
	assert.Contains(t, sshOut, "authenticated")
}

func runSetupKeys(topo string, targetURL string, home string) (string, error) {
	cmd := exec.Command(topo, "setup-keys", "--target", targetURL)
	cmd.Env = append(os.Environ(), "HOME="+home)
	if runtime.GOOS == "windows" {
		cmd.Env = append(cmd.Env, "USERPROFILE="+home)
	}
	cmd.Stdin = strings.NewReader("\n\n")

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func findGeneratedKey(t *testing.T, sshDir string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(sshDir, "id_ed25519_topo_*"))
	require.NoError(t, err)

	var keys []string
	for _, m := range matches {
		if !strings.HasSuffix(m, ".pub") {
			keys = append(keys, m)
		}
	}
	require.Len(t, keys, 1, "expected exactly one generated private key in %s, got: %v", sshDir, keys)
	return keys[0]
}

func readRemoteFile(t *testing.T, target *testutil.Container, path string) string {
	t.Helper()
	cmd := exec.Command("ssh", target.SSHDestination, "cat", path)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to read remote file %s: %s", path, out)
	return string(out)
}

func sshWithKey(destination string, keyPath string, args ...string) (string, error) {
	sshArgs := []string{
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-i", keyPath,
		destination,
	}
	sshArgs = append(sshArgs, args...)
	cmd := exec.Command("ssh", sshArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
