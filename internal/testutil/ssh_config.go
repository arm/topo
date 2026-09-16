package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const testSSHConfig = `Match host localhost user root
    UserKnownHostsFile ~/.ssh/topo-test-known-hosts/%p

Host *
`

func configureSSH(t testing.TB) {
	t.Helper()
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	sshDir := filepath.Join(home, ".ssh")
	require.NoError(t, os.MkdirAll(filepath.Join(sshDir, "topo-test-known-hosts"), 0o700))
	lock, err := os.OpenFile(filepath.Join(sshDir, "topo-test-config.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	defer func() { require.NoError(t, lock.Close()) }()
	require.NoError(t, lockFile(lock))

	path := filepath.Join(sshDir, "config")
	if _, err := os.Lstat(path); err == nil {
		path, err = filepath.EvalSymlinks(path)
		require.NoError(t, err)
	} else {
		require.True(t, os.IsNotExist(err), "locate SSH config: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		require.True(t, os.IsNotExist(err), "read SSH config: %v", err)
	}
	if bytes.Contains(content, []byte(testSSHConfig)) {
		return
	}

	file, err := os.CreateTemp(filepath.Dir(path), ".topo-test-config-*")
	require.NoError(t, err)
	defer func() {
		if err := os.Remove(file.Name()); !os.IsNotExist(err) { // #nosec G703 -- removes our temporary file.
			require.NoError(t, err)
		}
	}()
	_, writeErr := file.Write(append([]byte(testSSHConfig), content...))
	closeErr := file.Close()
	require.NoError(t, writeErr)
	require.NoError(t, closeErr)
	if info, err := os.Stat(path); err == nil {
		require.NoError(t, os.Chmod(file.Name(), info.Mode().Perm())) // #nosec G703 -- preserves permissions on our temporary file.
	} else {
		require.True(t, os.IsNotExist(err), "stat SSH config: %v", err)
	}
	require.NoError(t, os.Rename(file.Name(), path)) // #nosec G703 -- installs our temporary file at the resolved config path.
}
