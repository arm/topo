package ssh_test

import (
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestination(t *testing.T) {
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
					d := ssh.Destination(input)

					assert.True(t, d.IsPlainLocalhost())
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
					d := ssh.Destination(input)

					assert.False(t, d.IsPlainLocalhost())
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
					d := ssh.Destination(input)

					assert.False(t, d.IsPlainLocalhost())
				})
			}
		})
	})

	t.Run("IsLocalhost", func(t *testing.T) {
		t.Run("returns true for plain localhost", func(t *testing.T) {
			tests := []string{
				"localhost",
				"LOCALHOST",
				"LocalHost",
			}

			for _, input := range tests {
				t.Run(input, func(t *testing.T) {
					d := ssh.Destination(input)

					assert.True(t, d.IsLocalhost())
				})
			}
		})

		t.Run("returns true when user or port specified", func(t *testing.T) {
			tests := []string{
				"user@localhost",
				"user@127.0.0.1",
				"ssh://localhost:2222",
				"ssh://root@localhost:2222",
			}

			for _, input := range tests {
				t.Run(input, func(t *testing.T) {
					d := ssh.Destination(input)

					assert.True(t, d.IsLocalhost())
				})
			}
		})
	})

	t.Run("AsURI", func(t *testing.T) {
		t.Run("returns uri form of host string", func(t *testing.T) {
			d := ssh.Destination("user@host")

			assert.Equal(t, "ssh://user@host", d.AsURI())
		})

		t.Run("doesn't duplicate ssh:// scheme", func(t *testing.T) {
			d := ssh.Destination("ssh://user@host:123")

			assert.Equal(t, "ssh://user@host:123", d.AsURI())
		})
	})

	t.Run("Slugify", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"user@example.com", "user_example.com"},
			{"Example-Host", "Example-Host"},
			{"spaces and/tabs", "spaces_and_tabs"},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				require.Equal(t, tt.want, ssh.Destination(tt.input).Slugify(), "Slugify should replace special characters with underscores and keep allowed characters")
			})
		}
	})
}
