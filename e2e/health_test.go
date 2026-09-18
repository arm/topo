package e2e

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	container := testutil.StartContainer(t, testutil.DinDContainer)
	topo := buildBinary(t)

	t.Run("accurately shows host health status", func(t *testing.T) {
		out, err := runCheckHealth(topo, container)
		require.NoError(t, err)

		assert.Contains(t, out, " ✓ OpenSSH (ssh)")
		assert.Contains(t, out, " ✓ Container Engine (docker)")
		assert.Contains(t, out, " ✓ Docker Compose (docker compose)")
	})

	t.Run("shows that it's connected to a valid target", func(t *testing.T) {
		out, err := runCheckHealth(topo, container)
		require.NoError(t, err)

		assert.Contains(t, out, " ✓ Connectivity")
	})

	t.Run("fails to connect to an invalid target", func(t *testing.T) {
		fakeContainer := testutil.Container{
			SSHDestination: "fake@target",
			Name:           "fake-tgt-container",
		}
		out, err := runCheckHealth(topo, &fakeContainer)
		assert.NoError(t, err)
		assert.Contains(t, out, " ✗ Connectivity")
	})

	t.Run("outputs JSON when specified", func(t *testing.T) {
		out, err := runCheckHealth(topo, container, "--output", "json")

		assert.NoError(t, err)
		assert.Contains(t, out, container.SSHDestination)
		out = strings.ReplaceAll(out, container.SSHDestination, "TARGET_DESTINATION")
		testutil.AssertJsonGoldenFile(t, out, "testdata/TestHealthCheckJson.golden")
	})
}

func runCheckHealth(topo string, target *testutil.Container, args ...string) (string, error) {
	args = append([]string{"health", "--target", target.SSHDestination}, args...)
	healthCmd := exec.Command(topo, args...)

	out, err := healthCmd.CombinedOutput()
	return string(out), err
}
