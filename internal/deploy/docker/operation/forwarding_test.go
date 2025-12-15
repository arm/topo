package operation_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureTCPForwarding(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("returns expected string", func(t *testing.T) {
			forwarding := operation.NewVMHostRegistryBridge()

			got := forwarding.Description()

			assert.Equal(t, "Ensure TCP forwarding container exists", got)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints pull and run commands", func(t *testing.T) {
			var buf bytes.Buffer
			forwarding := operation.NewVMHostRegistryBridge()

			err := forwarding.DryRun(&buf)
			got := buf.String()

			require.NoError(t, err)
			assert.Contains(t, got, "docker pull alpine/socat:1.8.0.3")
			assert.Contains(t, got, "docker run -d --restart=unless-stopped --name=topo-registry-routing --network=host alpine/socat:1.8.0.3")
			assert.Contains(t, got, fmt.Sprintf("TCP-LISTEN:%d,fork,reuseaddr", ssh.RegistryPort))
			assert.Contains(t, got, fmt.Sprintf("TCP:host.docker.internal:%d", ssh.RegistryPort))
		})
	})
}
