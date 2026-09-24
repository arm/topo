package ssh_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustCreateSshDirectory(t *testing.T) string {
	tmp := t.TempDir()
	testutil.RequireMkdirAll(t, filepath.Join(tmp, ".ssh"))
	return filepath.Join(tmp, ".ssh")
}

func TestCreateOrModifyConfigFile(t *testing.T) {
	t.Run("writes include directive to default config file", func(t *testing.T) {
		sshDir := mustCreateSshDirectory(t)
		dest := ssh.Destination{Host: "board1"}
		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/id_ed25519"),
		}

		err := ssh.CreateOrModifyConfigFile(sshDir, dest, modifiers)
		require.NoError(t, err)

		configPath := filepath.Join(sshDir, ssh.DefaultConfigFileName)
		topoConfigPath := filepath.ToSlash(filepath.Join(sshDir, ssh.TopoConfigFileName))
		testutil.AssertFileContents(t, `Include `+topoConfigPath+`
`, configPath)
	})

	t.Run("creates topo-managed config file if it does not exist", func(t *testing.T) {
		sshDir := mustCreateSshDirectory(t)
		dest := ssh.Destination{Host: "board1"}
		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("User", "homer"),
		}

		err := ssh.CreateOrModifyConfigFile(sshDir, dest, modifiers)
		require.NoError(t, err)

		topoConfigPath := filepath.Join(sshDir, ssh.TopoConfigFileName)
		testutil.AssertFileContents(t, `Host board1
User homer
`, topoConfigPath)
	})

	t.Run("does not duplicate include directive in default config file if it already exists", func(t *testing.T) {
		sshDir := mustCreateSshDirectory(t)
		dest := ssh.Destination{Host: "board1"}
		modifiers := []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/id_ed25519"),
		}
		err := ssh.CreateOrModifyConfigFile(sshDir, dest, modifiers)
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(sshDir, dest, modifiers)

		wantConfig := fmt.Sprintf(`Include %s
`, filepath.ToSlash(filepath.Join(sshDir, ssh.TopoConfigFileName)))
		require.NoError(t, err)
		testutil.AssertFileContents(t, wantConfig, filepath.Join(sshDir, ssh.DefaultConfigFileName))
	})

	t.Run("adds new entry to existing topo-managed config file", func(t *testing.T) {
		sshDir := mustCreateSshDirectory(t)
		err := ssh.CreateOrModifyConfigFile(sshDir,
			ssh.Destination{Host: "board1"},
			[]ssh.ConfigDirectiveModifier{ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key1")},
		)
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(sshDir,
			ssh.Destination{Host: "board2"},
			[]ssh.ConfigDirectiveModifier{ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key2")},
		)
		require.NoError(t, err)

		topoConfigPath := filepath.Join(sshDir, ssh.TopoConfigFileName)
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
		sshDir := mustCreateSshDirectory(t)
		dest := ssh.Destination{Host: "board1"}
		err := ssh.CreateOrModifyConfigFile(sshDir, dest, []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key_old"),
			ssh.NewEnsureConfigDirective("User", "homer"),
		})
		require.NoError(t, err)

		err = ssh.CreateOrModifyConfigFile(sshDir, dest, []ssh.ConfigDirectiveModifier{
			ssh.NewEnsureConfigDirective("IdentityFile", "~/.ssh/key_new"),
		})
		require.NoError(t, err)

		topoConfigPath := filepath.Join(sshDir, ssh.TopoConfigFileName)
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
	t.Run("detects if legacy config directory does not exist", func(t *testing.T) {
		sshDir := mustCreateSshDirectory(t)

		isLegacyDir, err := ssh.IsLegacyTopoConfigDirectory(sshDir)

		assert.NoError(t, err)
		assert.False(t, isLegacyDir)
	})

	t.Run("detects if legacy config directory exists", func(t *testing.T) {
		sshDir := mustCreateSshDirectory(t)
		testutil.RequireMkdirAll(t, filepath.Join(sshDir, ssh.TopoConfigFileName))

		isLegacyDir, err := ssh.IsLegacyTopoConfigDirectory(sshDir)

		assert.NoError(t, err)
		assert.True(t, isLegacyDir)
	})
}

func TestGetConfigDirectory(t *testing.T) {
	t.Run("returns path to .ssh directory in user's home directory", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		got, err := ssh.GetConfigDirectory()

		require.NoError(t, err)
		require.Equal(t, filepath.Join(homeDir, ".ssh"), got)
	})
}
