package docker_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	deploytestutil "github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dinDContainer            = gtestutil.DinDContainer
	passwordlessSSHContainer = gtestutil.PasswordlessSSHContainer
)

func requireDocker(t *testing.T) {
	t.Helper()
	gtestutil.RequireDocker(t)
}

func requireLinuxDockerEngine(t *testing.T) {
	t.Helper()
	gtestutil.RequireLinuxDockerEngine(t)
}

func requireAvailableTCPPort(t *testing.T, host string) string {
	t.Helper()
	return gtestutil.RequireAvailableTCPPort(t, host)
}

func requireWriteFile(t *testing.T, path, content string) {
	t.Helper()
	gtestutil.RequireWriteFile(t, path, content)
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

func forceComposeDown(t *testing.T, composeFilePath string) {
	t.Helper()
	// #nosec G204 -- ignore as its a test helper
	err := exec.Command("docker", "compose", "-f", composeFilePath, "down", "-v").Run()
	if err != nil {
		t.Logf("docker compose down failed: %v (compose file: %s)", err, composeFilePath)
	}
}

func assertContainersRunning(t *testing.T, destination ssh.Destination, composeFilePath string) {
	t.Helper()
	dockerCommand := docker.ComposeCommand(t.Context(), docker.NewHostFromDestination(destination), composeFilePath, "ps", "--format", "json")
	output, err := dockerCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers running")

	containers, err := deploytestutil.UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Name"], container["State"])
	}
}

func assertContainersStopped(t *testing.T, destination ssh.Destination, composeFilePath string) {
	t.Helper()
	dockerCommand := docker.ComposeCommand(t.Context(), docker.NewHostFromDestination(destination), composeFilePath, "ps", "--format", "json", "--all")
	output, err := dockerCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers reported")

	containers, err := deploytestutil.UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "exited", container["State"], "expected container %s to be exited (state=%s)", container["Name"], container["State"])
	}
}
