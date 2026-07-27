package testutil

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	RequireDocker            = gtestutil.RequireDocker
	RequireLinuxDockerEngine = gtestutil.RequireLinuxDockerEngine
	RequireAvailableTCPPort  = gtestutil.RequireAvailableTCPPort
	RequireWriteFile         = gtestutil.RequireWriteFile
	SanitiseTestName         = gtestutil.SanitiseTestName
	StartContainer           = gtestutil.StartContainer
	DinDContainer            = gtestutil.DinDContainer
	PasswordlessSSHContainer = gtestutil.PasswordlessSSHContainer
)

func TestImageName(t *testing.T) string {
	return "test-image-" + gtestutil.SanitiseTestName(t)
}

func TestContainerName(t *testing.T) string {
	return "test-container-" + gtestutil.SanitiseTestName(t)
}

func TestProjectName(t *testing.T) string {
	return "test-project-" + gtestutil.SanitiseTestName(t)
}

func RequireImageExists(t *testing.T, h docker.Host, imageName string) {
	t.Helper()
	inspectCmd := docker.Docker(h, "image", "inspect", imageName)
	output, err := inspectCmd.CombinedOutput()
	require.NoError(t, err, "image %s doesn't exist: %s output: %s", imageName, docker.String(inspectCmd), string(output))
}

func RequireImageDoesNotExist(t *testing.T, h docker.Host, imageName string) {
	t.Helper()
	listCmd := docker.Docker(h, "image", "ls", "--quiet", "--filter", "reference="+imageName)
	output, err := listCmd.CombinedOutput()
	require.NoError(t, err, "failed to list image %s: %s output: %s", imageName, docker.String(listCmd), string(output))
	require.Empty(t, strings.TrimSpace(string(output)), "image %s unexpectedly exists", imageName)
}

func ForceComposeDown(t *testing.T, composeFilePath string) {
	t.Helper()
	// #nosec G204 -- ignore as its a test helper
	err := exec.Command("docker", "compose", "-f", composeFilePath, "down", "-v").Run()
	if err != nil {
		t.Logf("docker compose down failed: %v (compose file: %s)", err, composeFilePath)
	}
}

func AssertContainersRunning(t *testing.T, dest ssh.Destination, composeFilePath string) {
	t.Helper()
	dockerCmd := docker.DockerCompose(docker.NewHostFromDestination(dest), composeFilePath, "ps", "--format", "json")
	output, err := dockerCmd.CombinedOutput()
	require.NoError(t, err, string(output))

	require.NotEmpty(t, bytes.TrimSpace(output), "no containers running")

	containers, err := UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Name"], container["State"])
	}
}

func AssertContainersStopped(t *testing.T, dest ssh.Destination, composeFilePath string) {
	t.Helper()
	dockerCmd := docker.DockerCompose(docker.NewHostFromDestination(dest), composeFilePath, "ps", "--format", "json", "--all")
	output, err := dockerCmd.CombinedOutput()
	require.NoError(t, err, string(output))

	require.NotEmpty(t, bytes.TrimSpace(output), "no containers reported")

	containers, err := UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "exited", container["State"], "expected container %s to be exited (state=%s)", container["Name"], container["State"])
	}
}
