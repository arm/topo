package e2e

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const targetDestinationPlaceholder = "TARGET_DESTINATION"

func replaceNonDeterministicDestination(t *testing.T, out string) string {
	t.Helper()
	var obj map[string]map[string]any
	err := json.Unmarshal([]byte(out), &obj)
	require.NoError(t, err)

	obj["target"]["destination"] = targetDestinationPlaceholder

	normalizedOut, err := json.MarshalIndent(obj, "", "  ")
	require.NoError(t, err)
	return string(normalizedOut)
}

func TestHealthCheck(t *testing.T) {
	container := testutil.StartContainer(t, testutil.DinDContainer)
	topo := buildBinary(t)

	t.Run("shows successful host and target health status", func(t *testing.T) {
		out, err := runCheckHealth(topo, container)
		require.NoError(t, err)

		assert.Contains(t, out, "┌─ Host ")
		assert.Contains(t, out, "┌─ Target: "+container.SSHDestination+" ")
		assert.Equal(t, 2, strings.Count(out, " ✓ All checks passed"))
	})

	t.Run("shows successful checks with ticks in verbose mode", func(t *testing.T) {
		out, err := runCheckHealth(topo, container, "--verbose")
		require.NoError(t, err)

		assert.Contains(t, out, " ✓ OpenSSH (ssh)")
		assert.Contains(t, out, " ✓ Container Engine (docker)")
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
		out = replaceNonDeterministicDestination(t, out)
		testutil.AssertJsonGoldenFile(t, out, "testdata/TestHealthCheckJson.golden")
	})
}

func runCheckHealth(topo string, target *testutil.Container, args ...string) (string, error) {
	args = append([]string{"health", "--target", target.SSHDestination}, args...)
	healthCmd := exec.Command(topo, args...)

	out, err := healthCmd.CombinedOutput()
	return string(out), err
}
