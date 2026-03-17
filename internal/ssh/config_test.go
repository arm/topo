package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigFromBytes(t *testing.T) {
	t.Run("parses hostname, user, and port", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
user homer
port 2222
`)

		got := NewConfigFromBytes(input)

		want := Config{host: "springfield.nuclear.gov", user: "homer", port: "2222"}
		assert.Equal(t, want, got)
	})

	t.Run("ignores unrecognised keys", func(t *testing.T) {
		input := []byte(`hostname springfield.nuclear.gov
identityfile ~/.ssh/id_ed25519
user homer
`)

		got := NewConfigFromBytes(input)

		want := Config{host: "springfield.nuclear.gov", user: "homer"}
		assert.Equal(t, want, got)
	})

	t.Run("returns empty config for empty input", func(t *testing.T) {
		got := NewConfigFromBytes([]byte{})

		want := Config{}
		assert.Equal(t, want, got)
	})

	t.Run("matching is case-insensitive", func(t *testing.T) {
		input := []byte(`Hostname kwik.e.mart
User apu
Port 22
`)

		got := NewConfigFromBytes(input)

		want := Config{host: "kwik.e.mart", user: "apu", port: "22"}
		assert.Equal(t, want, got)
	})
}
