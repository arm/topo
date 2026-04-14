package ssh_test

import (
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
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
}

func TestResolveConfiguredUser(t *testing.T) {
	t.Run("with explicit configuration", func(t *testing.T) {
		explicitConfig := []byte(`debug1: /tmp/.ssh/topo_config line 1: Applying options for board-alias
debug1: /etc/ssh/ssh_config line 57: Applying options for *
user root
hostname 10.2.2.26
`)

		t.Run("returns destination's user when no user is configured", func(t *testing.T) {
			config := []byte(`debug1: /tmp/.ssh/topo_config line 1: Applying options for board-alias
debug1: /etc/ssh/ssh_config line 57: Applying options for *
hostname 10.2.2.26
`)
			dest := ssh.Destination{User: "root", Host: "board-alias"}

			got, err := ssh.ResolveConfiguredUser(dest, config)

			assert.NoError(t, err)
			assert.Equal(t, "root", got)
		})

		t.Run("returns destination's user when user is configured as the same user", func(t *testing.T) {
			dest := ssh.Destination{User: "root", Host: "board-alias"}

			got, err := ssh.ResolveConfiguredUser(dest, explicitConfig)

			assert.NoError(t, err)
			assert.Equal(t, "root", got)
		})

		t.Run("returns configured user when destination's user is not set", func(t *testing.T) {
			dest := ssh.Destination{Host: "board-alias"}

			got, err := ssh.ResolveConfiguredUser(dest, explicitConfig)

			assert.NoError(t, err)
			assert.Equal(t, "root", got)
		})

		t.Run("errors if destination's user doesn't match configured user", func(t *testing.T) {
			dest := ssh.Destination{User: "admin", Host: "board-alias"}

			_, err := ssh.ResolveConfiguredUser(dest, explicitConfig)

			assert.ErrorContains(t, err, `ssh host/alias "board-alias" is already associated with user "root"`)
		})
	})

	t.Run("without explicit host configuration", func(t *testing.T) {
		wildcardConfig := []byte(`debug1: /etc/ssh/ssh_config line 57: Applying options for *
user username
hostname board-alias
`)

		t.Run("returns destination's user when it's set", func(t *testing.T) {
			dest := ssh.Destination{User: "root", Host: "board-alias"}

			got, err := ssh.ResolveConfiguredUser(dest, wildcardConfig)

			assert.NoError(t, err)
			assert.Equal(t, "root", got)
		})

		t.Run("returns configured user when destination's user is not set", func(t *testing.T) {
			dest := ssh.Destination{Host: "board-alias"}

			got, err := ssh.ResolveConfiguredUser(dest, wildcardConfig)

			assert.NoError(t, err)
			assert.Equal(t, "username", got)
		})
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
