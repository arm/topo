package sshconfig_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/setupkeys/sshconfig"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestModifySSHConfig(t *testing.T) {
	t.Run("writes include and fragment", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		targetHost := "user@example.com:2222"
		targetFileName := "user_example_com_2222"
		privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetFileName))

		err := sshconfig.ModifySSHConfig(targetHost, privKeyPath, targetFileName, false, nil)
		require.NoError(t, err)

		mainConfigPath := filepath.Join(tmp, ".ssh", "config")
		mainConfig, err := os.ReadFile(mainConfigPath)
		require.NoError(t, err)
		mainConfigText := string(mainConfig)
		require.Contains(t, mainConfigText, "Include ")
		require.Contains(t, mainConfigText, filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config", "*.conf")))

		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))
		fragment, err := os.ReadFile(fragmentPath)
		require.NoError(t, err)

		expected := "" +
			"Host example.com\n" +
			"  HostName example.com\n" +
			"  User user\n" +
			"  Port 2222\n" +
			fmt.Sprintf("  IdentityFile %s\n", filepath.ToSlash(privKeyPath)) +
			"  IdentitiesOnly yes\n"
		require.Equal(t, expected, string(fragment))
	})

	t.Run("preserves existing fragment content", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		targetHost := "user@example.com:2222"
		targetFileName := "user_example_com_2222"
		privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetFileName))
		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))

		err := os.MkdirAll(filepath.Dir(fragmentPath), 0o700)
		require.NoError(t, err)

		existing := "" +
			"Host board-alias\n" +
			"  HostName example.com\n" +
			"  User vscode-user\n" +
			"  Port 2222\n"
		err = os.WriteFile(fragmentPath, []byte(existing), 0o600)
		require.NoError(t, err)

		err = sshconfig.ModifySSHConfig(targetHost, privKeyPath, targetFileName, false, nil)
		require.NoError(t, err)

		fragment, err := os.ReadFile(fragmentPath)
		require.NoError(t, err)

		expected := "" +
			"Host board-alias\n" +
			"  HostName example.com\n" +
			"  User vscode-user\n" +
			"  Port 2222\n" +
			fmt.Sprintf("  IdentityFile %s\n", filepath.ToSlash(privKeyPath)) +
			"  IdentitiesOnly yes\n"
		require.Equal(t, expected, string(fragment))
	})

	t.Run("updates existing key settings without replacing other fields", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		targetHost := "user@example.com:2222"
		targetFileName := "user_example_com_2222"
		privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetFileName))
		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))

		err := os.MkdirAll(filepath.Dir(fragmentPath), 0o700)
		require.NoError(t, err)

		existing := "" +
			"Host board-alias\n" +
			"  HostName example.com\n" +
			"  User vscode-user\n" +
			"  Port 2222\n" +
			"  IdentityFile /old/key\n" +
			"  IdentitiesOnly no\n"
		err = os.WriteFile(fragmentPath, []byte(existing), 0o600)
		require.NoError(t, err)

		err = sshconfig.ModifySSHConfig(targetHost, privKeyPath, targetFileName, false, nil)
		require.NoError(t, err)

		fragment, err := os.ReadFile(fragmentPath)
		require.NoError(t, err)

		expected := "" +
			"Host board-alias\n" +
			"  HostName example.com\n" +
			"  User vscode-user\n" +
			"  Port 2222\n" +
			fmt.Sprintf("  IdentityFile %s\n", filepath.ToSlash(privKeyPath)) +
			"  IdentitiesOnly yes\n"
		require.Equal(t, expected, string(fragment))
	})

	t.Run("deduplicates existing owned key settings", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		targetHost := "user@example.com:2222"
		targetFileName := "user_example_com_2222"
		privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetFileName))
		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))

		err := os.MkdirAll(filepath.Dir(fragmentPath), 0o700)
		require.NoError(t, err)

		existing := "" +
			"Host board-alias\n" +
			"  HostName example.com\n" +
			"  IdentityFile /old/key1\n" +
			"  User vscode-user\n" +
			"  IdentityFile /old/key2\n" +
			"  IdentitiesOnly no\n" +
			"  Port 2222\n"
		err = os.WriteFile(fragmentPath, []byte(existing), 0o600)
		require.NoError(t, err)

		err = sshconfig.ModifySSHConfig(targetHost, privKeyPath, targetFileName, false, nil)
		require.NoError(t, err)

		fragment, err := os.ReadFile(fragmentPath)
		require.NoError(t, err)

		expected := "" +
			"Host board-alias\n" +
			"  HostName example.com\n" +
			"  User vscode-user\n" +
			"  Port 2222\n" +
			fmt.Sprintf("  IdentityFile %s\n", filepath.ToSlash(privKeyPath)) +
			"  IdentitiesOnly yes\n"
		require.Equal(t, expected, string(fragment))
	})
}
