package testutil

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func DockerDeploymentFixture(t *testing.T) (project.Scope, string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	imageName := TestImageName(t)
	composeFilePath := gtestutil.RequireWriteComposeFile(t, temporaryDirectory, fmt.Sprintf(`
name: %s
services:
  a-service:
    build: .
    image: %s
    stop_grace_period: 1s
`, TestProjectName(t), imageName))
	gtestutil.RequireWriteFile(t, filepath.Join(temporaryDirectory, "Dockerfile"), `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		removeOutput, err := docker.Command(
			context.Background(), docker.LocalHost, "image", "rm", "-f", imageName,
		).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return project.Scope{ComposeFile: composeFilePath}, imageName
}

func RequireDockerImageExists(t *testing.T, host docker.Host, imageName string) {
	t.Helper()
	inspectCommand := docker.Command(t.Context(), host, "image", "inspect", imageName)
	output, err := inspectCommand.CombinedOutput()
	require.NoError(t, err, "image %s doesn't exist: %s", imageName, string(output))
}

func RequireDockerImageDoesNotExist(t *testing.T, host docker.Host, imageName string) {
	t.Helper()
	listCommand := docker.Command(t.Context(), host, "image", "ls", "--quiet", "--filter", "reference="+imageName)
	output, err := listCommand.CombinedOutput()
	require.NoError(t, err, "failed to list image %s: %s", imageName, string(output))
	require.Empty(t, strings.TrimSpace(string(output)), "image %s unexpectedly exists", imageName)
}

func RequireDockerRegistryContainerAbsent(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := docker.Command(t.Context(), docker.LocalHost, "inspect", containerName)
	require.Error(t, inspectCommand.Run(), "container %s already exists", containerName)
	t.Cleanup(func() {
		removeOutput, err := docker.Command(
			context.Background(), docker.LocalHost, "rm", "-f", containerName,
		).CombinedOutput()
		if err != nil && !strings.Contains(string(removeOutput), "No such container") {
			t.Logf("failed to remove registry container: %v: %s", err, string(removeOutput))
		}
	})
}

func CleanupDockerComposeProject(t *testing.T, scope project.Scope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := docker.ComposeCommand(ctx, docker.LocalHost, scope, "down", "-v")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("docker compose down failed: %v (compose file: %s): %s", err, scope.ComposeFile, output)
	}
}

func AssertDockerContainersRunning(t *testing.T, destination ssh.Destination, scope project.Scope) {
	t.Helper()
	dockerCommand := docker.ComposeCommand(
		t.Context(), docker.NewHostFromDestination(destination), scope, "ps", "--format", "json",
	)
	output, err := dockerCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers running")

	containers, err := UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "running", container["State"],
			"container %s is not running: %s", container["Name"], container["State"])
	}
}

func AssertDockerContainersStopped(t *testing.T, destination ssh.Destination, scope project.Scope) {
	t.Helper()
	dockerCommand := docker.ComposeCommand(
		t.Context(), docker.NewHostFromDestination(destination), scope, "ps", "--format", "json", "--all",
	)
	output, err := dockerCommand.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers reported")

	containers, err := UnmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "exited", container["State"],
			"expected container %s to be exited (state=%s)", container["Name"], container["State"])
	}
}
