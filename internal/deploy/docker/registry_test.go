package docker_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureRegistryRunning(t *testing.T) {
	requireLinuxDockerEngine(t)

	t.Run("creates a registry when its container does not exist", func(t *testing.T) {
		const containerName = "topo-test-registry-create"
		requireContainerAbsent(t, containerName)
		port := requireAvailableTCPPort(t, "127.0.0.1")
		var output bytes.Buffer

		err := docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertContainerRunning(t, containerName)
		assertContainerPort(t, containerName, port)
	})

	t.Run("starts an existing stopped registry", func(t *testing.T) {
		const containerName = "topo-test-registry-start"
		requireContainerAbsent(t, containerName)
		port := requireAvailableTCPPort(t, "127.0.0.1")
		var output bytes.Buffer
		require.NoError(t, docker.EnsureRegistryRunning(t.Context(), &output, containerName, port), output.String())
		stopOutput, err := docker.Command(t.Context(), docker.LocalHost, "stop", containerName).CombinedOutput()
		require.NoError(t, err, string(stopOutput))
		output.Reset()

		err = docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertContainerRunning(t, containerName)
		assertContainerPort(t, containerName, port)
	})

	t.Run("reuses a running registry with an engine-assigned port", func(t *testing.T) {
		const containerName = "topo-test-registry-assigned-port"
		port := startTestRegistry(t, containerName)
		var output bytes.Buffer

		err := docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.NoError(t, err, output.String())
		assertContainerRunning(t, containerName)
		assertContainerPort(t, containerName, port)
	})

	t.Run("returns an error when an existing registry uses a different port", func(t *testing.T) {
		const containerName = "topo-test-registry-port-mismatch"
		alreadyRunningOnPort := startTestRegistry(t, containerName)
		newlyRequestedPort := "5000"
		if newlyRequestedPort == alreadyRunningOnPort {
			newlyRequestedPort = "5001"
		}
		var output bytes.Buffer

		err := docker.EnsureRegistryRunning(t.Context(), &output, containerName, newlyRequestedPort)

		require.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("registry port mismatch (running: %s, requested: %s)", alreadyRunningOnPort, newlyRequestedPort))
		assert.ErrorContains(t, err, fmt.Sprintf("docker rm -f %s", containerName))
	})

	t.Run("adds a diagnostic when the registry port is already in use", func(t *testing.T) {
		const containerName = "topo-test-registry-port-conflict"
		requireContainerAbsent(t, containerName)
		const portOwnerContainerName = "topo-test-registry-port-owner"
		port := startTestRegistry(t, portOwnerContainerName)
		var output bytes.Buffer

		err := docker.EnsureRegistryRunning(t.Context(), &output, containerName, port)

		require.Error(t, err)
		assert.ErrorContains(t, err, fmt.Sprintf("port is already in use, this could be an existing %s or another process", containerName))
	})
}

func startTestRegistry(t *testing.T, containerName string) string {
	t.Helper()
	requireContainerAbsent(t, containerName)
	output, err := docker.Command(t.Context(), docker.LocalHost,
		"run", "-d", "-p", "127.0.0.1::5000", "--name", containerName, "registry:2",
	).CombinedOutput()
	require.NoError(t, err, string(output))
	output, err = docker.Command(t.Context(), docker.LocalHost, "port", containerName, "5000").CombinedOutput()
	require.NoError(t, err, string(output))
	host, port, err := net.SplitHostPort(strings.TrimSpace(string(output)))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host)
	return port
}

func requireContainerAbsent(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := docker.Command(t.Context(), docker.LocalHost, "inspect", containerName)
	require.Error(t, inspectCommand.Run(), "container %s already exists", containerName)
	t.Cleanup(func() {
		removeOutput, err := docker.Command(context.Background(), docker.LocalHost, "rm", "-f", containerName).CombinedOutput()
		if err != nil && !strings.Contains(string(removeOutput), "No such container") {
			t.Logf("failed to remove registry container: %v: %s", err, string(removeOutput))
		}
	})
}

func assertContainerRunning(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := docker.Command(t.Context(), docker.LocalHost, "inspect", "--format", "{{.State.Running}}", containerName)
	output, err := inspectCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "true", strings.TrimSpace(string(output)))
}

func assertContainerPort(t *testing.T, containerName, port string) {
	t.Helper()
	portCommand := docker.Command(t.Context(), docker.LocalHost, "port", containerName, "5000")
	output, err := portCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	assert.Equal(t, "127.0.0.1:"+port, strings.TrimSpace(string(output)))
}
