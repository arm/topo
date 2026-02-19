package ssh_test

import (
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestHost(t *testing.T) {
	t.Run("IsPlainLocalhost", func(t *testing.T) {
		t.Run("returns true for plain localhost", func(t *testing.T) {
			tests := []string{
				"localhost",
				"LOCALHOST",
				"LocalHost",
				"127.0.0.1",
			}

			for _, input := range tests {
				t.Run(input, func(t *testing.T) {
					h := ssh.Host(input)

					assert.True(t, h.IsPlainLocalhost())
				})
			}
		})

		t.Run("returns false when user or port specified", func(t *testing.T) {
			tests := []string{
				"user@localhost",
				"user@127.0.0.1",
				"localhost:2222",
				"user@localhost:2222",
			}

			for _, input := range tests {
				t.Run(input, func(t *testing.T) {
					h := ssh.Host(input)

					assert.False(t, h.IsPlainLocalhost())
				})
			}
		})

		t.Run("returns false for remote hosts", func(t *testing.T) {
			tests := []string{
				"remote",
				"user@remote",
				"user@remote:2222",
			}

			for _, input := range tests {
				t.Run(input, func(t *testing.T) {
					h := ssh.Host(input)

					assert.False(t, h.IsPlainLocalhost())
				})
			}
		})
	})

	t.Run("AsURI", func(t *testing.T) {
		t.Run("returns uri form of host string", func(t *testing.T) {
			h := ssh.Host("user@host")

			assert.Equal(t, "ssh://user@host", h.AsURI())
		})
	})
}
