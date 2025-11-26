package ssh_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestHost(t *testing.T) {
	t.Run("AsURI", func(t *testing.T) {
		t.Run("returns uri form of host string", func(t *testing.T) {
			h := ssh.Host("user@host")

			assert.Equal(t, "ssh://user@host", h.AsURI())
		})
	})
}
