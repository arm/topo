package ssh_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestSSHAlias(t *testing.T) {
	t.Run("ControlSocketPath", func(t *testing.T) {
		t.Run("returns deterministic path", func(t *testing.T) {
			h := ssh.SSHAlias{Alias: "happy_alias"}

			got := h.ControlSocketPath()

			hash := sha256.Sum256([]byte("happy_alias"))
			want := filepath.Join(os.TempDir(), fmt.Sprintf("topo-tunnel-%x", hash[:8]))
			assert.Equal(t, want, got)
		})
	})

	t.Run("FormatSSHConnectCommand", func(t *testing.T) {
		t.Run("includes control socket and port when enabled", func(t *testing.T) {
			h := ssh.SSHAlias{Alias: "jolly_alias", Port: "2222"}

			got := strings.Join(h.FormatSSHConnectCommand(true, "5000"), " ")

			want := fmt.Sprintf(
				"ssh -N -o ExitOnForwardFailure=yes -p 2222 -fMS %s -R 5000:127.0.0.1:5000 jolly_alias",
				h.ControlSocketPath(),
			)
			assert.Equal(t, want, got)
		})

		t.Run("omits control socket when disabled", func(t *testing.T) {
			h := ssh.SSHAlias{Alias: "ecstatic_alias"}

			got := strings.Join(h.FormatSSHConnectCommand(false, "5000"), " ")

			want := "ssh -N -o ExitOnForwardFailure=yes -R 5000:127.0.0.1:5000 ecstatic_alias"
			assert.Equal(t, want, got)
		})
	})

	t.Run("FormatSSHExitCommand", func(t *testing.T) {
		t.Run("includes control socket and port", func(t *testing.T) {
			h := ssh.SSHAlias{Alias: "bouncy_alias", Port: "2222"}

			got := strings.Join(h.FormatSSHExitCommand(), " ")

			want := fmt.Sprintf("ssh -p 2222 -S %s -O exit bouncy_alias", h.ControlSocketPath())
			assert.Equal(t, want, got)
		})
	})

	t.Run("GetHost", func(t *testing.T) {
		h := ssh.SSHAlias{Host: "glum.com"}

		assert.Equal(t, "glum.com", h.GetHost())
	})

	t.Run("IsLocalhost", func(t *testing.T) {
		t.Run("returns true for localhost hosts", func(t *testing.T) {
			tests := []ssh.SSHAlias{
				{Host: "localhost"},
				{Host: "LOCALHOST"},
				{Host: "127.0.0.1"},
			}

			for _, input := range tests {
				t.Run(input.Host, func(t *testing.T) {
					assert.True(t, input.IsLocalhost())
				})
			}
		})

		t.Run("returns false for remote hosts", func(t *testing.T) {
			tests := []ssh.SSHAlias{
				{Host: "grin_alias"},
				{Host: "sad.com"},
			}

			for _, input := range tests {
				t.Run(input.Host, func(t *testing.T) {
					assert.False(t, input.IsLocalhost())
				})
			}
		})
	})

	t.Run("IsAlias", func(t *testing.T) {
		t.Run("returns true for plain alias", func(t *testing.T) {
			assert.True(t, ssh.IsAlias("gleeful_alias"))
		})

		t.Run("returns false for explicit ssh hosts", func(t *testing.T) {
			tests := []string{
				"grinner@grinning_alias",
				"grinning_alias:2222",
				"ssh://skipper@skipping_alias",
				"[2001:db8::1]",
			}

			for _, input := range tests {
				t.Run(input, func(t *testing.T) {
					assert.False(t, ssh.IsAlias(input))
				})
			}
		})
	})
}
