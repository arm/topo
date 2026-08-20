package podman_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureRegistryRunning(t *testing.T) {
	requireLocalPodman(t)

	t.Run("creates a registry when its container does not exist", func(t *testing.T) {
		const containerName = "topo-test-registry-create"
		requireRegistryContainerAbsent(t, containerName)
		port := requireAvailableTCPPort(t)
		var output bytes.Buffer

		err := podman.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertRegistryContainerRunning(t, containerName)
		assertRegistryContainerPort(t, containerName, port)
	})

	t.Run("starts an existing stopped registry", func(t *testing.T) {
		const containerName = "topo-test-registry-start"
		requireRegistryContainerAbsent(t, containerName)
		port := requireAvailableTCPPort(t)
		var output bytes.Buffer
		require.NoError(t, podman.EnsureRegistryRunning(t.Context(), &output, containerName, port), output.String())
		stopOutput, err := podman.Command(t.Context(), podman.LocalSocket, "stop", containerName).CombinedOutput()
		require.NoError(t, err, string(stopOutput))
		output.Reset()

		err = podman.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertRegistryContainerRunning(t, containerName)
		assertRegistryContainerPort(t, containerName, port)
	})

	t.Run("returns an error when an existing registry uses a different port", func(t *testing.T) {
		const containerName = "topo-test-registry-port-mismatch"
		requireRegistryContainerAbsent(t, containerName)
		alreadyRunningOnPort := requireAvailableTCPPort(t)
		newlyRequestedPort := requireAvailableTCPPort(t)
		for newlyRequestedPort == alreadyRunningOnPort {
			newlyRequestedPort = requireAvailableTCPPort(t)
		}
		var output bytes.Buffer
		require.NoError(t, podman.EnsureRegistryRunning(t.Context(), &output, containerName, alreadyRunningOnPort), output.String())
		output.Reset()

		err := podman.EnsureRegistryRunning(t.Context(), &output, containerName, newlyRequestedPort)

		require.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("registry port mismatch (running: %s, requested: %s)", alreadyRunningOnPort, newlyRequestedPort))
		assert.ErrorContains(t, err, fmt.Sprintf("podman rm -f %s", containerName))
	})

	t.Run("adds a diagnostic when the registry port is already in use", func(t *testing.T) {
		const containerName = "topo-test-registry-port-conflict"
		requireRegistryContainerAbsent(t, containerName)
		const portOwnerContainerName = "topo-test-registry-port-owner"
		requireRegistryContainerAbsent(t, portOwnerContainerName)
		port := requireAvailableTCPPort(t)
		portOwnerOutput, err := podman.Command(
			t.Context(),
			podman.LocalSocket,
			"run",
			"-d",
			"-p", fmt.Sprintf("127.0.0.1:%s:5000", port),
			"--name", portOwnerContainerName,
			"registry:2",
		).CombinedOutput()
		require.NoError(t, err, string(portOwnerOutput))
		var output bytes.Buffer

		err = podman.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("port is already in use, this could be an existing %s or another process", containerName))
	})
}

func requireRegistryContainerAbsent(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := podman.Command(t.Context(), podman.LocalSocket, "inspect", containerName)
	require.Error(t, inspectCommand.Run(), "container %s already exists", containerName)
	t.Cleanup(func() {
		removeOutput, err := podman.Command(context.Background(), podman.LocalSocket, "rm", "-f", containerName).CombinedOutput()
		if err != nil && !strings.Contains(string(removeOutput), "no container with name or ID") {
			t.Logf("failed to remove registry container: %v: %s", err, removeOutput)
		}
	})
}

func assertRegistryContainerRunning(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := podman.Command(t.Context(), podman.LocalSocket, "inspect", "--format", "{{.State.Running}}", containerName)
	output, err := inspectCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "true", strings.TrimSpace(string(output)))
}

func assertRegistryContainerPort(t *testing.T, containerName, port string) {
	t.Helper()
	portCommand := podman.Command(t.Context(), podman.LocalSocket, "port", containerName, "5000")
	output, err := portCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "127.0.0.1:"+port, strings.TrimSpace(string(output)))
}
