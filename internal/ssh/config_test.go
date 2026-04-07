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
