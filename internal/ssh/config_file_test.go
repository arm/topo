package ssh_test

import (
	"fmt"
	"path/filepath"
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
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh"))
		dest := ssh.Destination{Host: "board1"}
		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/id_ed25519"),
		}

		err := ssh.CreateOrModifyConfigFile(dest, modifiers)
		require.NoError(t, err)

		configPath := filepath.Join(tmp, ".ssh", "config")
		topoConfigPath := filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config"))
		testutil.AssertFileContents(t, `Include `+topoConfigPath+`
`, configPath)
	})

	t.Run("creates topo-managed config file if it does not exist", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh"))
		dest := ssh.Destination{Host: "board1"}
		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("User", "homer"),
		}

		err := ssh.CreateOrModifyConfigFile(dest, modifiers)
		require.NoError(t, err)

		topoConfigPath := filepath.Join(tmp, ".ssh", "topo_config")
		testutil.AssertFileContents(t, `Host board1
User homer
`, topoConfigPath)
	})

	t.Run("does not duplicate include directive in default config file if it already exists", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh"))
		dest := ssh.Destination{Host: "board1"}
		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/id_ed25519"),
		}
		err := ssh.CreateOrModifyConfigFile(dest, modifiers)
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(dest, modifiers)

		wantConfig := fmt.Sprintf(`Include %s
`, filepath.ToSlash(filepath.Join(tmp, ".ssh", "topo_config")))
		require.NoError(t, err)
		testutil.AssertFileContents(t, wantConfig, filepath.Join(tmp, ".ssh", "config"))
	})

	t.Run("adds new entry to existing topo-managed config file", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh"))
		err := ssh.CreateOrModifyConfigFile(
			ssh.Destination{Host: "board1"},
			[]ssh.ConfigDirectiveModifier{ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key1")},
		)
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(
			ssh.Destination{Host: "board2"},
			[]ssh.ConfigDirectiveModifier{ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key2")},
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
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh"))
		dest := ssh.Destination{Host: "board1"}
		err := ssh.CreateOrModifyConfigFile(dest, []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key_old"),
			ssh.NewEnsureConfigDirective("User", "homer"),
		})
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(dest, []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key_new"),
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
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh", "topo_config"))

		err := ssh.CheckForLegacyTopoConfigEntries()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "legacy topo ssh config")
	})
}

func TestMigrateLegacyConfig(t *testing.T) {
	t.Run("returns error when no legacy topo config directory exists", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)

		err := ssh.MigrateLegacyTopoConfig()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nothing to migrate")
	})

	t.Run("returns error when legacy directory has no conf files", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh", "topo_config"))

		err := ssh.MigrateLegacyTopoConfig()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no .conf files")
	})

	t.Run("concatenates conf files into unified config and removes directory", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		legacyDir := filepath.Join(tmp, ".ssh", "topo_config")
		legacyDirSlash := filepath.ToSlash(legacyDir)
		board1Conf := `Host board1
  IdentityFile ~/.ssh/key1
`
		board2Conf := `Host board2
  IdentityFile ~/.ssh/key2
`
		sshConf := "Include " + legacyDirSlash + `/*.conf

Host *
`
		testutil.RequireMkdirAll(t, legacyDir)
		testutil.RequireWriteFile(t, filepath.Join(legacyDir, "topo_board1.conf"), board1Conf)
		testutil.RequireWriteFile(t, filepath.Join(legacyDir, "topo_board2.conf"), board2Conf)
		testutil.RequireWriteFile(t, filepath.Join(tmp, ".ssh", "config"), sshConf)

		require.NoError(t, ssh.MigrateLegacyTopoConfig())

		mergedFile := legacyDirSlash
		wantSshConfAfterMigration := fmt.Sprintf(`
Include %s
Host *
`, mergedFile)
		testutil.AssertFileContents(t, board1Conf+board2Conf, mergedFile)
		testutil.AssertFileContents(t, wantSshConfAfterMigration, filepath.Join(tmp, ".ssh", "config"))
	})

	t.Run("adds include directive if ssh config does not exist", func(t *testing.T) {
		tmp := t.TempDir()
		testutil.SetHomeDir(t, tmp)
		legacyDir := filepath.Join(tmp, ".ssh", "topo_config")
		testutil.RequireMkdirAll(t, legacyDir)
		testutil.RequireWriteFile(t, filepath.Join(legacyDir, "topo_board1.conf"), `Host board1
  IdentityFile ~/.ssh/key1
`)

		err := ssh.MigrateLegacyTopoConfig()

		wantConfig := fmt.Sprintf(`Include %s
`, filepath.ToSlash(legacyDir))
		require.NoError(t, err)
		testutil.AssertFileContents(t, wantConfig, filepath.Join(tmp, ".ssh", "config"))
	})
}
