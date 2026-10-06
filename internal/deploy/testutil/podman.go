package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RequirePodmanRegistryContainerAbsent(t *testing.T, containerName string) {
	t.Helper()
	inspectCommand := podman.Command(t.Context(), podman.LocalSocket, "inspect", containerName)
	require.Error(t, inspectCommand.Run(), "container %s already exists", containerName)
	t.Cleanup(func() {
		removeOutput, err := podman.Command(
			context.Background(), podman.LocalSocket, "rm", "-f", containerName,
		).CombinedOutput()
		if err != nil && !strings.Contains(string(removeOutput), "no container with name or ID") {
			t.Logf("failed to remove registry container: %v: %s", err, removeOutput)
		}
	})
}

func PodmanDeploymentFixture(t *testing.T) (project.Scope, string) {
	t.Helper()
	tempDir := t.TempDir()
	testName := gtestutil.SanitiseTestName(t)
	imageName := "test-image-" + testName
	composeFileContent := fmt.Sprintf(`
name: %s
services:
  built:
    build: .
    image: %s
    stop_grace_period: 1s
  pulled:
    image: docker.io/library/alpine:latest
    command: ["tail", "-f", "/dev/null"]
    stop_grace_period: 1s
`, "test-project-"+testName, imageName)
	composeFileContent, err := gtestutil.FixPodmanInDockerQuirk(composeFileContent)
	require.NoError(t, err)
	composeFile := gtestutil.RequireWriteComposeFile(t, tempDir, composeFileContent)
	gtestutil.RequireWriteFile(t, filepath.Join(tempDir, "Dockerfile"), `
FROM docker.io/library/alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		removeOutput, err := podman.Command(ctx, podman.LocalSocket, "image", "rm", "-f", imageName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return project.Scope{ComposeFile: composeFile}, "test-project-" + testName
}

func CleanupPodmanComposeProject(t *testing.T, scope project.Scope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd, err := podman.ComposeCommand(ctx, podman.LocalSocket, scope, "down", "-v", "--remove-orphans", "--rmi", "local")
	if err != nil {
		t.Logf("failed to configure Podman Compose: %v", err)
		return
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("Podman Compose cleanup failed: %v: %s", err, output)
	}
}

func AssertPodmanContainersInState(t *testing.T, projectName string, socket podman.Socket, state string) {
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
		assert.Equal(t, state, container["State"],
			"expected container %s to be %s (state=%s)", container["Names"], state, container["State"])
	}
}
