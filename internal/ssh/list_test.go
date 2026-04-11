package ssh_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSSHConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return filepath.ToSlash(path)
}

func TestListHosts(t *testing.T) {
	t.Run("returns hosts from a single config file", func(t *testing.T) {
		tmp := t.TempDir()
		configPath := writeSSHConfig(t, tmp, "config", `
Host board1
  HostName 192.168.0.1

Host board2
  HostName 192.168.0.2
`)

		got := ssh.ListHosts(configPath)

		assert.ElementsMatch(t, []string{"board1", "board2"}, got)
	})

	t.Run("excludes wildcard host pattern", func(t *testing.T) {
		tmp := t.TempDir()
		configPath := writeSSHConfig(t, tmp, "config", `
Host *
  ServerAliveInterval 60

Host myhost
  HostName 10.0.0.1
`)

		got := ssh.ListHosts(configPath)

		assert.ElementsMatch(t, []string{"myhost"}, got)
	})

	t.Run("follows include directives", func(t *testing.T) {
		tmp := t.TempDir()
		includedPath := writeSSHConfig(t, tmp, "extra_config", `
Host included-host
  HostName 10.0.0.2
`)
		configPath := writeSSHConfig(t, tmp, "config", fmt.Sprintf(`
Include %s

Host main-host
  HostName 10.0.0.1
`, includedPath))

		got := ssh.ListHosts(configPath)

		assert.ElementsMatch(t, []string{"main-host", "included-host"}, got)
	})

	t.Run("deduplicates hosts across files", func(t *testing.T) {
		tmp := t.TempDir()
		includedPath := writeSSHConfig(t, tmp, "extra_config", `
Host shared-host
  HostName 10.0.0.1
`)
		configPath := writeSSHConfig(t, tmp, "config", `
Include `+includedPath+`

Host shared-host
  HostName 10.0.0.1
`)

		got := ssh.ListHosts(configPath)

		assert.ElementsMatch(t, []string{"shared-host"}, got)
	})

	t.Run("handles cyclic includes without infinite loop", func(t *testing.T) {
		tmp := t.TempDir()
		configAPath := filepath.Join(tmp, "config_a")
		configBPath := filepath.Join(tmp, "config_b")

		writeSSHConfig(t, tmp, "config_a", `
Include `+configBPath+`

Host host-a
  HostName 10.0.0.1
`)
		writeSSHConfig(t, tmp, "config_b", `
Include `+configAPath+`

Host host-b
  HostName 10.0.0.2
`)

		got := ssh.ListHosts(configAPath)

		assert.Nil(t, got, "should return without hanging on cyclic includes")
	})

	t.Run("returns empty slice for nonexistent config file", func(t *testing.T) {
		got := ssh.ListHosts("/nonexistent/path/config")

		assert.Empty(t, got)
	})

	t.Run("returns empty slice for config with only wildcard host", func(t *testing.T) {
		tmp := t.TempDir()
		configPath := writeSSHConfig(t, tmp, "config", `
Host *
  ServerAliveInterval 60
`)

		got := ssh.ListHosts(configPath)

		assert.Empty(t, got)
	})
}
