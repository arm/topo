package ssh_test

import (
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDestination(t *testing.T) {
	t.Run("valid ssh destinations", func(t *testing.T) {
		cases := []struct {
			raw  string
			want ssh.Destination
		}{
			{
				raw:  "example.com",
				want: ssh.Destination{Host: "example.com"},
			},
			{
				raw:  "user@example.com",
				want: ssh.Destination{User: "user", Host: "example.com"},
			},
			{
				raw:  "ssh://example.com",
				want: ssh.Destination{Host: "example.com"},
			},
			{
				raw:  "ssh://user@example.com",
				want: ssh.Destination{User: "user", Host: "example.com"},
			},
			{
				raw:  "ssh://example.com:2222",
				want: ssh.Destination{Host: "example.com", Port: "2222"},
			},
			{
				raw:  "ssh://user@example.com:2222",
				want: ssh.Destination{User: "user", Host: "example.com", Port: "2222"},
			},
			{
				raw:  "ssh://[2001:db8::1]",
				want: ssh.Destination{Host: "2001:db8::1"},
			},
			{
				raw:  "ssh://[2001:db8::1]:2222",
				want: ssh.Destination{Host: "2001:db8::1", Port: "2222"},
			},
		}

		for _, tc := range cases {
			t.Run(tc.raw, func(t *testing.T) {
				got := ssh.NewDestination(tc.raw)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("convenience destinations", func(t *testing.T) {
		cases := []struct {
			raw  string
			want ssh.Destination
		}{
			{
				raw:  "example.com:2222",
				want: ssh.Destination{Host: "example.com", Port: "2222"},
			},
			{
				raw:  "user@example.com:2222",
				want: ssh.Destination{User: "user", Host: "example.com", Port: "2222"},
			},
			{
				raw:  "[2001:db8::1]",
				want: ssh.Destination{Host: "2001:db8::1"},
			},
			{
				raw:  "user@[2001:db8::1]:2222",
				want: ssh.Destination{User: "user", Host: "2001:db8::1", Port: "2222"},
			},
		}

		for _, tc := range cases {
			t.Run(tc.raw, func(t *testing.T) {
				got := ssh.NewDestination(tc.raw)
				assert.Equal(t, tc.want, got)
			})
		}
	})
}

func TestDestination(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		t.Run("returns the uri form of destination", func(t *testing.T) {
			tests := []struct {
				desc string
				sut  ssh.Destination
				want string
			}{
				{
					desc: "just host",
					sut:  ssh.Destination{Host: "localhost"},
					want: "ssh://localhost",
				},
			}

			for _, test := range tests {
				t.Run(test.desc, func(t *testing.T) {
					assert.Equal(t, test.want, test.sut.String())
				})
			}
		})
	})

	t.Run("IsPlainLocalhost", func(t *testing.T) {
		t.Run("returns true if host is localhost", func(t *testing.T) {
			tests := []string{
				"localhost",
				"LOCALHOST",
				"LocalHost",
				"127.0.0.1",
			}

			for _, host := range tests {
				t.Run(host, func(t *testing.T) {
					d := ssh.Destination{Host: host}

					assert.True(t, d.IsPlainLocalhost())
				})
			}
		})

		t.Run("returns false when user or port specified", func(t *testing.T) {
			tests := []struct {
				desc string
				sut  ssh.Destination
			}{
				{
					desc: "user specified",
					sut:  ssh.Destination{User: "obi-wan", Host: "death-star"},
				},
				{
					desc: "port specified",
					sut:  ssh.Destination{Host: "death-star", Port: "hole-you-shoot-into"},
				},
			}

			for _, test := range tests {
				t.Run(test.desc, func(t *testing.T) {
					assert.False(t, test.sut.IsPlainLocalhost())
				})
			}
		})

		t.Run("returns false for remote hosts", func(t *testing.T) {
			d := ssh.Destination{Host: "remote"}

			assert.False(t, d.IsPlainLocalhost())
		})
	})

	t.Run("IsLocalhost", func(t *testing.T) {
		t.Run("returns true if host is localhost", func(t *testing.T) {
			tests := []struct {
				desc string
				sut  ssh.Destination
			}{
				{
					desc: "case insensitive",
					sut:  ssh.Destination{Host: "LoCaLhOsT"},
				},
				{
					desc: "user specified",
					sut:  ssh.Destination{User: "leet-hacker", Host: "127.0.0.1"},
				},
				{
					desc: "port specified",
					sut:  ssh.Destination{Host: "localhost", Port: "1337"},
				},
			}

			for _, test := range tests {
				t.Run(test.desc, func(t *testing.T) {
					assert.True(t, test.sut.IsLocalhost())
				})
			}
		})
	})

	t.Run("AsURI", func(t *testing.T) {
		t.Run("returns uri form of the destination", func(t *testing.T) {
			d := ssh.Destination{
				User: "darth-vader",
				Host: "death-star",
				Port: "deep-breath",
			}

			got := d.AsURI()

			want := "ssh://darth-vader@death-star:deep-breath"
			assert.Equal(t, want, got)
		})
	})

	t.Run("Slugify", func(t *testing.T) {
		t.Run("slugifies the uri", func(t *testing.T) {
			d := ssh.Destination{
				User: "darth-vader",
				Host: "death-star",
				Port: "deep-breath",
			}

			got := d.Slugify()

			want := "darth-vader_death-star_deep-breath"
			assert.Equal(t, want, got)
		})
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
			{raw: "ssh://user@example.com:2222", wantUser: "user", wantHost: "example.com", wantPort: "2222"},
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
