package ssh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigFromBytes(t *testing.T) {
	t.Run("parses basic config fields", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
user homer
`)

		got := NewConfigFromBytes(input)

		want := Config{
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

		got := NewConfigFromBytes(input)

		want := Config{
			HostName: "springfield.nuclear.gov",
			User:     "homer",
		}
		assert.Equal(t, want, got)
	})

	t.Run("returns empty config for empty input", func(t *testing.T) {
		got := NewConfigFromBytes([]byte{})

		want := Config{}
		assert.Equal(t, want, got)
	})

	t.Run("matching is case-insensitive", func(t *testing.T) {
		input := []byte(`HoStNaMe kwik.e.mart`)

		got := NewConfigFromBytes(input)

		want := Config{
			HostName: "kwik.e.mart",
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses connecttimeout as duration", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
connecttimeout 30
`)

		got := NewConfigFromBytes(input)

		assert.Equal(t, 30*time.Second, got.connectTimeout)
	})

	t.Run("ignores non-numeric connecttimeout", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
connecttimeout none
`)

		got := NewConfigFromBytes(input)

		assert.Equal(t, time.Duration(0), got.connectTimeout)
	})
}

func TestConfigConnectTimeout(t *testing.T) {
	const fallback = 5 * time.Second

	t.Run("returns user config value when set", func(t *testing.T) {
		config := Config{connectTimeout: 30 * time.Second}

		assert.Equal(t, 30*time.Second, config.ConnectTimeout(fallback))
	})

	t.Run("returns fallback when not set in config", func(t *testing.T) {
		config := Config{}

		assert.Equal(t, fallback, config.ConnectTimeout(fallback))
	})
}

func TestIsExplicitHostConfig(t *testing.T) {
	t.Run("returns true for exact host matches in verbose ssh output", func(t *testing.T) {
		config := []byte(`debug1: /tmp/config line 1: Applying options for Board,board-alt
debug1: /tmp/config line 5: Applying options for *
hostname springfield.nuclear.gov
`)

		got := isExplicitHostConfig("board", config)
		assert.True(t, got)
	})

	t.Run("ignores negated host patterns", func(t *testing.T) {
		config := []byte(`debug1: /tmp/config line 1: Applying options for Board,!skip,*.corp,te?t
hostname springfield.nuclear.gov
`)

		got := isExplicitHostConfig("skip", config)
		assert.False(t, got)
	})

	t.Run("returns false when the host is not in the effective host list", func(t *testing.T) {
		config := []byte(`debug1: /tmp/config line 1: Applying options for board,board-alt
hostname springfield.nuclear.gov
`)

		got := isExplicitHostConfig("other-board", config)
		assert.False(t, got)
	})

	t.Run("ignores lines without an applying options marker", func(t *testing.T) {
		config := []byte(`hostname springfield.nuclear.gov
user homer
debug1: /tmp/config line 5: Applying options for *
`)

		got := isExplicitHostConfig("board", config)
		assert.False(t, got)
	})
}
