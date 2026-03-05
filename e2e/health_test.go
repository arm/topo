package e2e

import (
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	target := testutil.StartTargetContainer(t)
	topo := buildBinary(t)

	t.Run("accurately shows host health status", func(t *testing.T) {
		out, err := runCheckHealth(topo, target)
		require.NoError(t, err)

		assert.Contains(t, out, "SSH: ✅ (ssh)")
		assert.Contains(t, out, "Container Engine: ✅")
	})

	t.Run("shows that it's connected to a valid target", func(t *testing.T) {
		out, err := runCheckHealth(topo, target)
		require.NoError(t, err)

		assert.Contains(t, out, "Connected: ✅")
	})

	t.Run("fails to connect to an invalid target", func(t *testing.T) {
		fakeContainer := testutil.TargetContainer{
			SSHDestination: "fake@target",
			ContainerName:  "fake-tgt-container",
		}
		out, err := runCheckHealth(topo, &fakeContainer)
		assert.NoError(t, err)
		assert.Contains(t, out, "Connected: ❌")
	})
}

func runCheckHealth(topo string, target *testutil.TargetContainer) (string, error) {
	args := []string{"health", "--target", target.SSHDestination}
	healthCmd := exec.Command(topo, args...)

	out, err := healthCmd.CombinedOutput()
	return string(out), err
}
