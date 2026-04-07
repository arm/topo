package ssh_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrModifyConfigFile(t *testing.T) {
	t.Run("writes include directive to default config file", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".ssh"), 0o700))
		dest := ssh.Destination{Host: "board1"}
		directives := []ssh.ConfigDirective{
			ssh.NewConfigDirective("IdentityFile", "~/.ssh/id_ed25519"),
		}

		err := ssh.CreateOrModifyConfigFile(dest, directives)
		require.NoError(t, err)

		configPath := filepath.Join(tmp, ".ssh", "config")
		topoConfigPath := filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config"))
		testutil.AssertFileContents(t, `Include `+topoConfigPath+`
`, configPath)
	})

	t.Run("creates topo-managed config file if it does not exist", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".ssh"), 0o700))
		dest := ssh.Destination{Host: "board1"}
		directives := []ssh.ConfigDirective{
			ssh.NewConfigDirective("User", "homer"),
		}

		err := ssh.CreateOrModifyConfigFile(dest, directives)
		require.NoError(t, err)

		topoConfigPath := filepath.Join(tmp, ".ssh", "topo_config")
		testutil.AssertFileContents(t, `Host board1
User homer
`, topoConfigPath)
	})

	t.Run("does not duplicate include directive in default config file if it already exists", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".ssh"), 0o700))
		dest := ssh.Destination{Host: "board1"}
		directives := []ssh.ConfigDirective{
			ssh.NewConfigDirective("IdentityFile", "~/.ssh/id_ed25519"),
		}
		err := ssh.CreateOrModifyConfigFile(dest, directives)
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(dest, directives)

		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(tmp, ".ssh", "config"))
		require.NoError(t, err)
		count := strings.Count(string(got), "Include")
		assert.Equal(t, 1, count, `Include directive should appear exactly once, got:
%s`, got)
	})

	t.Run("adds new entry to existing topo-managed config file", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".ssh"), 0o700))
		err := ssh.CreateOrModifyConfigFile(
			ssh.Destination{Host: "board1"},
			[]ssh.ConfigDirective{ssh.NewConfigDirective("IdentityFile", "~/.ssh/key1")},
		)
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(
			ssh.Destination{Host: "board2"},
			[]ssh.ConfigDirective{ssh.NewConfigDirective("IdentityFile", "~/.ssh/key2")},
		)
		require.NoError(t, err)

		topoConfigPath := filepath.Join(tmp, ".ssh", "topo_config")
		testutil.AssertFileContents(t,
			`Host board1
IdentityFile ~/.ssh/key1
Host board2
IdentityFile ~/.ssh/key2
`,
			topoConfigPath,
		)
	})

	t.Run("modifies existing entry in topo-managed config file, preserving unmodified directives", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".ssh"), 0o700))
		dest := ssh.Destination{Host: "board1"}
		err := ssh.CreateOrModifyConfigFile(dest, []ssh.ConfigDirective{
			ssh.NewConfigDirective("IdentityFile", "~/.ssh/key_old"),
			ssh.NewConfigDirective("User", "homer"),
		})
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(dest, []ssh.ConfigDirective{
			ssh.NewConfigDirective("IdentityFile", "~/.ssh/key_new"),
		})
		require.NoError(t, err)

		topoConfigPath := filepath.Join(tmp, ".ssh", "topo_config")
		testutil.AssertFileContents(t,
			`Host board1
IdentityFile ~/.ssh/key_new
User homer
`,
			topoConfigPath,
		)
	})
}

func TestCheckForLegacyConfigEntries(t *testing.T) {
	t.Run("returns nil if no legacy config file exists", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		err := ssh.CheckForLegacyTopoConfigEntries()

		assert.NoError(t, err)
	})

	t.Run("returns error if legacy config directory exists", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		err := os.MkdirAll(filepath.Join(tmp, ".ssh"), 0o700)
		require.NoError(t, err)
		err = os.Mkdir(filepath.Join(tmp, ".ssh", "topo_config"), 0o600)
		require.NoError(t, err)

		err = ssh.CheckForLegacyTopoConfigEntries()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "legacy topo ssh config")
	})
}
