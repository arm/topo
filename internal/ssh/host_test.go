package ssh_test

import (
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		t.Run("doesn't duplicate ssh:// scheme", func(t *testing.T) {
			h := ssh.Host("ssh://user@host:123")

			assert.Equal(t, "ssh://user@host:123", h.AsURI())
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
				require.Equal(t, tt.want, ssh.Host(tt.input).Slugify(), "Slugify should replace special characters with underscores and keep allowed characters")
			})
		}
	})

	t.Run("SplitUserHostPort", func(t *testing.T) {
		cases := []struct {
			raw      string
			wantUser string
			wantHost string
			wantPort string
		}{
			{raw: "user@example.com:2222", wantUser: "user", wantHost: "example.com", wantPort: "2222"},
			{raw: "example.com:2222", wantUser: "", wantHost: "example.com", wantPort: "2222"},
			{raw: "example.com", wantUser: "", wantHost: "example.com", wantPort: ""},
			{raw: "user@example.com", wantUser: "user", wantHost: "example.com", wantPort: ""},
			{raw: "[2001:db8::1]", wantUser: "", wantHost: "2001:db8::1", wantPort: ""},
			{raw: "user@[2001:db8::1]:2222", wantUser: "user", wantHost: "2001:db8::1", wantPort: "2222"},
			{raw: "[2001:db8::1]:2222", wantUser: "", wantHost: "2001:db8::1", wantPort: "2222"},
		}

		for _, tc := range cases {
			t.Run(tc.raw, func(t *testing.T) {
				user, host, port := ssh.SplitUserHostPort(tc.raw)
				require.Equal(t, tc.wantUser, user, "user for %q", tc.raw)
				require.Equal(t, tc.wantHost, host, "host for %q", tc.raw)
				require.Equal(t, tc.wantPort, port, "port for %q", tc.raw)
			})
		}
	})
}
