package ssh_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHTunnelStart(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			host := ssh.Host("user@remote")

			st := ssh.NewSSHTunnelStart(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -S %s -L %d:localhost:%d user@remote", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.Equal(t, want, got)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			host := ssh.Host("user@remote:2222")

			st := ssh.NewSSHTunnelStart(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -p 2222 -S %s -L %d:localhost:%d user@remote", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.Equal(t, want, got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("it outputs the ssh command", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.Host("user@remote")

			st := ssh.NewSSHTunnelStart(host)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -S %s -L %d:localhost:%d user@remote", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.Host("user@remote:2222")

			st := ssh.NewSSHTunnelStart(host)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -p 2222 -S %s -L %d:localhost:%d user@remote", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := ssh.NewSSHTunnelStart(ssh.Host("user@remote"))

			got := st.Description()

			assert.Equal(t, "Open registry SSH tunnel", got)
		})
	})
}

func TestSSHTunnelStartEdgeCases(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it handles host without user", func(t *testing.T) {
			host := ssh.Host("remote-server")

			st := ssh.NewSSHTunnelStart(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -S %s -L %d:localhost:%d remote-server", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.Equal(t, want, got)
		})

		t.Run("it handles host without user but with port", func(t *testing.T) {
			host := ssh.Host("remote-server:2222")

			st := ssh.NewSSHTunnelStart(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -p 2222 -S %s -L %d:localhost:%d remote-server", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.Equal(t, want, got)
		})

		t.Run("it handles IP address", func(t *testing.T) {
			host := ssh.Host("user@192.168.1.100")
			want := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -S %s -L %d:localhost:%d user@192.168.1.100", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			st := ssh.NewSSHTunnelStart(host)

			got := strings.Join(st.Command().Args, " ")

			assert.Equal(t, want, got)
		})

		t.Run("it handles IP address with port", func(t *testing.T) {
			host := ssh.Host("user@192.168.1.100:2222")

			st := ssh.NewSSHTunnelStart(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -MNf -o ExitOnForwardFailure=yes -p 2222 -S %s -L %d:localhost:%d user@192.168.1.100", ssh.ControlSocketPath(string(host)), ssh.RegistryPort, ssh.RegistryPort)
			assert.Equal(t, want, got)
		})
	})
}

func TestSSHTunnelStop(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("it generates correct ssh command", func(t *testing.T) {
			host := ssh.Host("user@remote")

			st := ssh.NewSSHTunnelStop(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -S %s -O exit user@remote", ssh.ControlSocketPath(string(host)))
			assert.Equal(t, want, got)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			host := ssh.Host("user@remote:2222")

			st := ssh.NewSSHTunnelStop(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit user@remote", ssh.ControlSocketPath(string(host)))
			assert.Equal(t, want, got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("generates the correct ssh command", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.Host("user@remote")

			st := ssh.NewSSHTunnelStop(host)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -S %s -O exit user@remote", ssh.ControlSocketPath(string(host)))
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})

		t.Run("it includes port flag when host has custom port", func(t *testing.T) {
			var buf bytes.Buffer
			host := ssh.Host("user@remote:2222")

			st := ssh.NewSSHTunnelStop(host)
			err := st.DryRun(&buf)
			got := strings.TrimSpace(buf.String())

			require.NoError(t, err)
			wantSuffix := fmt.Sprintf("ssh -p 2222 -S %s -O exit user@remote", ssh.ControlSocketPath(string(host)))
			assert.True(t, strings.HasSuffix(got, wantSuffix),
				"DryRun output %q does not end with %q", got, wantSuffix)
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			st := ssh.NewSSHTunnelStop(ssh.Host("user@remote"))

			got := st.Description()

			assert.Equal(t, "Close registry SSH tunnel", got)
		})
	})
}

func TestSSHTunnelStopEdgeCases(t *testing.T) {
	t.Run("Command", func(t *testing.T) {
		t.Run("handles host without user", func(t *testing.T) {
			host := ssh.Host("remote-server")

			st := ssh.NewSSHTunnelStop(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -S %s -O exit remote-server", ssh.ControlSocketPath(string(host)))
			assert.Equal(t, want, got)
		})

		t.Run("handles host without user but with port", func(t *testing.T) {
			host := ssh.Host("remote-server:2222")

			st := ssh.NewSSHTunnelStop(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit remote-server", ssh.ControlSocketPath(string(host)))
			assert.Equal(t, want, got)
		})

		t.Run("handles IP address", func(t *testing.T) {
			host := ssh.Host("user@192.168.1.100")

			st := ssh.NewSSHTunnelStop(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -S %s -O exit user@192.168.1.100", ssh.ControlSocketPath(string(host)))
			assert.Equal(t, want, got)
		})

		t.Run("handles IP address with port", func(t *testing.T) {
			host := ssh.Host("user@192.168.1.100:2222")

			st := ssh.NewSSHTunnelStop(host)
			got := strings.Join(st.Command().Args, " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit user@192.168.1.100", ssh.ControlSocketPath(string(host)))
			assert.Equal(t, want, got)
		})
	})
}
