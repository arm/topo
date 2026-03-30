package ssh

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigFromBytes(t *testing.T) {
	t.Run("parses hostname", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
`)

		got := NewConfigFromBytes(input)

		want := Config{
			HostName: "springfield.nuclear.gov",
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses user", func(t *testing.T) {
		input := []byte(`user homer
`)

		got := NewConfigFromBytes(input)

		want := Config{
			User: "homer",
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

	t.Run("records exact host matches from verbose ssh output", func(t *testing.T) {
		input := []byte(`debug1: /tmp/config line 1: Applying options for board
debug1: /tmp/config line 5: Applying options for *
hostname springfield.nuclear.gov
user homer
`)

		got := NewConfigFromBytes(input)

		assert.True(t, got.HasExactHostMatch("board"))
		assert.False(t, got.HasExactHostMatch("other-board"))
	})

	t.Run("splits matched host patterns while ignoring negations", func(t *testing.T) {
		input := []byte(`debug1: /tmp/config line 1: Applying options for board,!skip,board-alt
`)

		got := NewConfigFromBytes(input)

		assert.True(t, got.HasExactHostMatch("board"))
		assert.True(t, got.HasExactHostMatch("board-alt"))
		assert.False(t, got.HasExactHostMatch("skip"))
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
