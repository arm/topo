package podman_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sanitiseTestName(t *testing.T) string {
	t.Helper()
	return gtestutil.SanitiseTestName(t)
}

func startPodmanInContainer(t *testing.T) *gtestutil.Container {
	t.Helper()
	return gtestutil.StartContainer(t, gtestutil.PodmanContainer)
}

func assertContainersRunning(t *testing.T, projectName string, socket podman.Socket) {
	assertContainersInState(t, projectName, socket, "running")
}

func assertContainersStopped(t *testing.T, projectName string, socket podman.Socket) {
	assertContainersInState(t, projectName, socket, "exited")
}

func assertContainersInState(t *testing.T, projectName string, socket podman.Socket, state string) {
	t.Helper()
	cmd := podman.Command(t.Context(), socket,
		"ps", "--format", "json", "--all",
		"--filter", "label=com.docker.compose.project="+projectName,
	)
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	output, err := cmd.Output()
	require.NoError(t, err, "stdout: %s\nstderr: %s", output, diagnostics.String())

	var containers []map[string]any
	require.NoError(t, json.Unmarshal(output, &containers))
	require.NotEmpty(t, containers, "no containers reported; stderr: %s", diagnostics.String())

	for _, container := range containers {
		assert.Equal(t, state, container["State"], "expected container %s to be %s (state=%s)", container["Names"], state, container["State"])
	}
}

func startTestRegistry(t *testing.T, containerName string) string {
	t.Helper()
	requireRegistryContainerAbsent(t, containerName)
	output, err := podman.Command(t.Context(), podman.LocalSocket,
		"run", "-d", "-p", "127.0.0.1::5000", "--name", containerName, "registry:2",
	).CombinedOutput()
	require.NoError(t, err, string(output))
	output, err = podman.Command(t.Context(), podman.LocalSocket, "port", containerName, "5000").CombinedOutput()
	require.NoError(t, err, string(output))
	host, port, err := net.SplitHostPort(strings.TrimSpace(string(output)))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", host)
	return port
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
