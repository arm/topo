package docker_test

import (
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/docker"
	deploytestutil "github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var dinDContainer = gtestutil.DinDContainer

func requireDocker(t *testing.T) {
	t.Helper()
	gtestutil.RequireDocker(t)
}

func requireLinuxDockerEngine(t *testing.T) {
	t.Helper()
	gtestutil.RequireLinuxDockerEngine(t)
}

func startContainer(t *testing.T, spec gtestutil.ContainerSpec) *gtestutil.Container {
	t.Helper()
	return gtestutil.StartContainer(t, spec)
}

func testImageName(t *testing.T) string {
	return deploytestutil.TestImageName(t)
}

func testContainerName(t *testing.T) string {
	return deploytestutil.TestContainerName(t)
}

func testProjectName(t *testing.T) string {
	return deploytestutil.TestProjectName(t)
}

func requireImageExists(t *testing.T, host docker.Host, imageName string) {
	t.Helper()
	inspectCommand := docker.Command(t.Context(), host, "image", "inspect", imageName)
	output, err := inspectCommand.CombinedOutput()
	require.NoError(t, err, "image %s doesn't exist: %s", imageName, string(output))
}

func requireImageDoesNotExist(t *testing.T, host docker.Host, imageName string) {
	t.Helper()
	listCommand := docker.Command(t.Context(), host, "image", "ls", "--quiet", "--filter", "reference="+imageName)
	output, err := listCommand.CombinedOutput()
	require.NoError(t, err, "failed to list image %s: %s", imageName, string(output))
	require.Empty(t, strings.TrimSpace(string(output)), "image %s unexpectedly exists", imageName)
}

func forceComposeDown(t *testing.T, scope project.Scope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := docker.ComposeCommand(ctx, docker.LocalHost, scope, "down", "-v")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("docker compose down failed: %v (compose file: %s): %s", err, scope.ComposeFile, output)
	}
}

func assertContainersRunning(t *testing.T, destination ssh.Destination, scope project.Scope) {
	t.Helper()
	dockerCommand := docker.ComposeCommand(t.Context(), docker.NewHostFromDestination(destination), scope, "ps", "--format", "json")
	output, err := dockerCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers running")

	containers, err := deploytestutil.UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Name"], container["State"])
	}
}

func assertContainersStopped(t *testing.T, destination ssh.Destination, scope project.Scope) {
	t.Helper()
	dockerCommand := docker.ComposeCommand(t.Context(), docker.NewHostFromDestination(destination), scope, "ps", "--format", "json", "--all")
	output, err := dockerCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers reported")

	containers, err := deploytestutil.UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "exited", container["State"], "expected container %s to be exited (state=%s)", container["Name"], container["State"])
	}
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
