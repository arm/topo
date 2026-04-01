package ssh_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromBytes(t *testing.T) {
	t.Run("parses basic config fields", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
user homer
`)

		got := ssh.NewConfigFromBytes(input)

		want := ssh.Config{
			HostName: "springfield.nuclear.gov",
			User:     "homer",
		}
		assert.Equal(t, want, got)
	})

	t.Run("ignores unrecognised keys", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
identityfile ~/.ssh/id_ed25519
user homer
`)

		got := ssh.NewConfigFromBytes(input)

		want := ssh.Config{
			HostName: "springfield.nuclear.gov",
			User:     "homer",
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns empty config for empty input", func(t *testing.T) {
		got := ssh.NewConfigFromBytes([]byte{})

		want := ssh.Config{}
		assert.Equal(t, want, got)
	})

	t.Run("matching is case-insensitive", func(t *testing.T) {
		input := []byte(`HoStNaMe kwik.e.mart`)

		got := ssh.NewConfigFromBytes(input)

		want := ssh.Config{
			HostName: "kwik.e.mart",
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses connecttimeout as duration", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
connecttimeout 30
`)

		got := ssh.NewConfigFromBytes(input)

		assert.Equal(t, 30*time.Second, got.ConnectTimeout(0))
	})

	t.Run("ignores non-numeric connecttimeout", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
connecttimeout none
`)

		got := ssh.NewConfigFromBytes(input)

		assert.Equal(t, time.Duration(0), got.ConnectTimeout(0))
	})
}

func TestConfigConnectTimeout(t *testing.T) {
	const fallback = 5 * time.Second

	t.Run("returns user config value when set", func(t *testing.T) {
		configContent := []byte(`connecttimeout 30
hostname springfield.nuclear.gov
`)
		config := ssh.NewConfigFromBytes(configContent)

		assert.Equal(t, 30*time.Second, config.ConnectTimeout(fallback))
	})

	t.Run("returns fallback when not set in config", func(t *testing.T) {
		config := ssh.Config{}

		assert.Equal(t, fallback, config.ConnectTimeout(fallback))
	})
}

func TestIsExplicitHostConfig(t *testing.T) {
	t.Run("returns true for exact host matches in verbose ssh output", func(t *testing.T) {
		config := []byte(`debug1: /tmp/config line 1: Applying options for Board,board-alt
debug1: /tmp/config line 5: Applying options for *
hostname springfield.nuclear.gov
`)

		got := ssh.IsExplicitHostConfig("board", config)
		assert.True(t, got)
	})

	t.Run("ignores negated host patterns", func(t *testing.T) {
		config := []byte(`debug1: /tmp/config line 1: Applying options for Board,!skip,*.corp,te?t
hostname springfield.nuclear.gov
`)

		got := ssh.IsExplicitHostConfig("skip", config)
		assert.False(t, got)
	})

	t.Run("returns false when the host is not in the effective host list", func(t *testing.T) {
		config := []byte(`debug1: /tmp/config line 1: Applying options for board,board-alt
hostname springfield.nuclear.gov
`)

		got := ssh.IsExplicitHostConfig("other-board", config)
		assert.False(t, got)
	})

	t.Run("ignores lines without an applying options marker", func(t *testing.T) {
		config := []byte(`hostname springfield.nuclear.gov
user homer
debug1: /tmp/config line 5: Applying options for *
`)

		got := ssh.IsExplicitHostConfig("board", config)
		assert.False(t, got)
	})
}

func TestModifySSHConfig(t *testing.T) {
	t.Run("writes include and fragment", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		targetHost := ssh.NewDestination("user@example.com:2222")
		targetFileName := "user_example_com_2222"
		privKeyPath := filepath.Join(tmp, ".ssh", fmt.Sprintf("id_ed25519_topo_%s", targetFileName))

		err := ssh.CreateOrModifyConfig(targetHost, targetFileName, []ssh.ConfigDirective{
			ssh.NewDirectiveIdentityFile(privKeyPath),
			ssh.NewDirective("IdentitiesOnly", "yes"),
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

		targetHost := ssh.NewDestination("user@example.com:2222")
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

		err = ssh.CreateOrModifyConfig(targetHost, targetFileName, []ssh.ConfigDirective{
			ssh.NewDirectiveIdentityFile(privKeyPath),
			ssh.NewDirective("IdentitiesOnly", "yes"),
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

		targetHost := ssh.NewDestination("user@example.com:2222")
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

		err = ssh.CreateOrModifyConfig(targetHost, targetFileName, []ssh.ConfigDirective{
			ssh.NewDirectiveIdentityFile(privKeyPath),
			ssh.NewDirective("IdentitiesOnly", "yes"),
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

		targetHost := ssh.NewDestination("user@example.com:2222")
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

		err = ssh.CreateOrModifyConfig(targetHost, targetFileName, []ssh.ConfigDirective{
			ssh.NewDirectiveIdentityFile(privKeyPath),
			ssh.NewDirective("IdentitiesOnly", "yes"),
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

func TestCreateConfig(t *testing.T) {
	t.Run("handles creation of new ssh config file", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		targetHost := ssh.NewDestination("user@example.com:2222")
		targetFileName := "user_example_com_2222"
		fragmentPath := filepath.Join(tmp, ".ssh", "topo_config", fmt.Sprintf("topo_%s.conf", targetFileName))
		mainConfigPath := filepath.Join(tmp, ".ssh", "config")

		err := ssh.CreateConfig(targetHost, targetFileName)

		require.NoError(t, err)
		wantFragmentContents := "Host example.com\n  HostName example.com\n  User user\n  Port 2222\n"
		wantIncludedFragmentPath := filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config", "*.conf"))
		wantSSHConfigContents := fmt.Sprintf("Include %s\n\n", wantIncludedFragmentPath)
		testutil.AssertFileContents(t, wantFragmentContents, fragmentPath)
		testutil.AssertFileContents(t, wantSSHConfigContents, mainConfigPath)
	})
}
