package docker_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureRegistryRunning(t *testing.T) {
	testutil.RequireLinuxDockerEngine(t)

	t.Run("creates a registry when its container does not exist", func(t *testing.T) {
		const containerName = "topo-test-registry-create"
		requireContainerAbsent(t, containerName)
		port := testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		var output bytes.Buffer

		err := docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertRegistryRunning(t, containerName)
		assertRegistryPort(t, containerName, port)
	})

	t.Run("starts an existing stopped registry", func(t *testing.T) {
		const containerName = "topo-test-registry-start"
		requireContainerAbsent(t, containerName)
		port := testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		var output bytes.Buffer
		require.NoError(t, docker.EnsureRegistryRunning(t.Context(), &output, containerName, port), output.String())
		stopOutput, err := command.Docker(command.LocalHost, "stop", containerName).CombinedOutput()
		require.NoError(t, err, string(stopOutput))
		output.Reset()

		err = docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertRegistryRunning(t, containerName)
		assertRegistryPort(t, containerName, port)
	})

	t.Run("returns an error when an existing registry uses a different port", func(t *testing.T) {
		const containerName = "topo-test-registry-port-mismatch"
		requireContainerAbsent(t, containerName)
		alreadyRunningOnPort := testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		newlyRequestedPort := testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		for newlyRequestedPort == alreadyRunningOnPort {
			newlyRequestedPort = testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		}
		var output bytes.Buffer
		require.NoError(t, docker.EnsureRegistryRunning(t.Context(), &output, containerName, alreadyRunningOnPort), output.String())
		output.Reset()

		err := docker.EnsureRegistryRunning(t.Context(), &output, containerName, newlyRequestedPort)

		require.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("registry port mismatch (running: %s, requested: %s)", alreadyRunningOnPort, newlyRequestedPort))
		assert.ErrorContains(t, err, fmt.Sprintf("docker rm -f %s", containerName))
	})

	t.Run("adds a diagnostic when the registry port is already in use", func(t *testing.T) {
		const containerName = "topo-test-registry-port-conflict"
		requireContainerAbsent(t, containerName)
		const portOwnerContainerName = "topo-test-registry-port-owner"
		requireContainerAbsent(t, portOwnerContainerName)
		port := testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		portOwnerOutput, err := command.Docker(
			command.LocalHost,
			"run",
			"-d",
			"-p", fmt.Sprintf("127.0.0.1:%s:5000", port),
			"--name", portOwnerContainerName,
			"registry:2",
		).CombinedOutput()
		require.NoError(t, err, string(portOwnerOutput))
		var output bytes.Buffer

		err = docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("port is already in use, this could be an existing %s or another process", containerName))
	})
}

func requireContainerAbsent(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := command.Docker(command.LocalHost, "inspect", containerName)
	require.Error(t, inspectCommand.Run(), "container %s already exists", containerName)
	t.Cleanup(func() {
		removeOutput, err := command.Docker(command.LocalHost, "rm", "-f", containerName).CombinedOutput()
		if err != nil && !strings.Contains(string(removeOutput), "No such container") {
			t.Logf("failed to remove registry container: %v: %s", err, string(removeOutput))
		}
	})
}

func assertRegistryRunning(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := command.Docker(command.LocalHost, "inspect", "--format", "{{.State.Running}}", containerName)
	output, err := inspectCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "true", strings.TrimSpace(string(output)))
}

func assertRegistryPort(t *testing.T, containerName, port string) {
	t.Helper()
	portCommand := command.Docker(command.LocalHost, "port", containerName, "5000")
	output, err := portCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "127.0.0.1:"+port, strings.TrimSpace(string(output)))
}
