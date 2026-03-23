package sshconfig_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/ssh/sshconfig"
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

		err := sshconfig.CreateOrModifySSHConfig(targetHost, targetFileName, []sshconfig.SSHConfigDirective{
			sshconfig.NewDirectiveIdentityFile(privKeyPath),
			sshconfig.NewDirective("IdentitiesOnly", "yes"),
		})
		require.NoError(t, err)

		mainConfigPath := filepath.Join(tmp, ".ssh", "config")
		wantIncludedFragmentPath := filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config", "*.conf"))
		wantSSHConfigContents := fmt.Sprintf("Include %s\n\n", wantIncludedFragmentPath)
		testutil.AssertFileContents(t, wantSSHConfigContents, mainConfigPath)

		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))
		wantFragmentContents := fmt.Sprintf(`Host example.com
  HostName example.com
  User user
  Port 2222
  IdentityFile %s
  IdentitiesOnly yes
`, filepath.ToSlash(privKeyPath))
		testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
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

		existing := `Host board-alias
  HostName example.com
  User vscode-user
  Port 2222
`
		err = os.WriteFile(fragmentPath, []byte(existing), 0o600)
		require.NoError(t, err)

		err = sshconfig.CreateOrModifySSHConfig(targetHost, targetFileName, []sshconfig.SSHConfigDirective{
			sshconfig.NewDirectiveIdentityFile(privKeyPath),
			sshconfig.NewDirective("IdentitiesOnly", "yes"),
		})
		require.NoError(t, err)

		wantFragmentContents := fmt.Sprintf(`Host board-alias
  HostName example.com
  User vscode-user
  Port 2222
  IdentityFile %s
  IdentitiesOnly yes
`, filepath.ToSlash(privKeyPath))
		testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
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

		existing := `Host board-alias
  HostName example.com
  User vscode-user
  Port 2222
  IdentityFile /old/key
  IdentitiesOnly no
`
		err = os.WriteFile(fragmentPath, []byte(existing), 0o600)
		require.NoError(t, err)

		err = sshconfig.CreateOrModifySSHConfig(targetHost, targetFileName, []sshconfig.SSHConfigDirective{
			sshconfig.NewDirectiveIdentityFile(privKeyPath),
			sshconfig.NewDirective("IdentitiesOnly", "yes"),
		})
		require.NoError(t, err)

		wantFragmentContents := fmt.Sprintf(`Host board-alias
  HostName example.com
  User vscode-user
  Port 2222
  IdentityFile %s
  IdentitiesOnly yes
`, filepath.ToSlash(privKeyPath))
		testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
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

		existing := `Host board-alias
  HostName example.com
  IdentityFile /old/key1
  User vscode-user
  IdentityFile /old/key2
  IdentitiesOnly no
  Port 2222`
		err = os.WriteFile(fragmentPath, []byte(existing), 0o600)
		require.NoError(t, err)

		err = sshconfig.CreateOrModifySSHConfig(targetHost, targetFileName, []sshconfig.SSHConfigDirective{
			sshconfig.NewDirectiveIdentityFile(privKeyPath),
			sshconfig.NewDirective("IdentitiesOnly", "yes"),
		})
		require.NoError(t, err)

		wantFragmentContents := fmt.Sprintf(`Host board-alias
  HostName example.com
  User vscode-user
  Port 2222
  IdentityFile %s
  IdentitiesOnly yes
`, filepath.ToSlash(privKeyPath))
		testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
	})
}

func TestCreateSSHConfig(t *testing.T) {
	t.Run("handles creation of new ssh config file", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		targetHost := "user@example.com:2222"
		targetFileName := "user_example_com_2222"
		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))
		mainConfigPath := filepath.Join(tmp, ".ssh", "config")

		err := sshconfig.CreateSSHConfig(targetHost, targetFileName)

		require.NoError(t, err)
		wantFragmentContents := "Host example.com\n  HostName example.com\n  User user\n  Port 2222\n"
		wantIncludedFragmentPath := filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config", "*.conf"))
		wantSSHConfigContents := fmt.Sprintf("Include %s\n\n", wantIncludedFragmentPath)
		testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
		testutil.AssertFileContents(t, wantSSHConfigContents, mainConfigPath)
	})
}
