package podman_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	requireLocalPodman(t)
	composeFile, projectName := deploymentFixture(t)
	t.Cleanup(func() { cleanupComposeProject(t, composeFile) })

	err := podman.Deploy(t.Context(), io.Discard, composeFile)

	require.NoError(t, err)
	assertContainersRunning(t, projectName)
}

func requireLocalPodman(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman is not installed")
	}
	if _, err := exec.LookPath("docker-compose"); err != nil {
		t.Skip("docker-compose is not installed")
	}
	if output, err := exec.Command("podman", "info").CombinedOutput(); err != nil {
		t.Skipf("local Podman engine is unavailable: %v: %s", err, output)
	}
}

func assertContainersRunning(t *testing.T, projectName string) {
	t.Helper()
	cmd := podman.Command(t.Context(), "ps", "--format", "json", "--all",
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
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Names"], container["State"])
	}
}

func deploymentFixture(t *testing.T) (string, string) {
	t.Helper()
	tempDir := t.TempDir()
	testName := sanitiseTestName(t)
	imageName := "test-image-" + testName
	composeFile := filepath.Join(tempDir, "compose.yaml")
	composeFileContent := fmt.Sprintf(`
name: %s
services:
  built:
    build: .
    image: %s
  pulled:
    image: docker.io/library/alpine:latest
    command: ["tail", "-f", "/dev/null"]
`, "test-project-"+testName, imageName)
	requireWriteFile(t, composeFile, composeFileContent)
	requireWriteFile(t, filepath.Join(tempDir, "Dockerfile"), `
FROM docker.io/library/alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		removeOutput, err := podman.Command(ctx, "image", "rm", "-f", imageName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return composeFile, "test-project-" + testName
}

func cleanupComposeProject(t *testing.T, composeFile string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := podman.ComposeCommand(ctx, composeFile, "down", "-v", "--remove-orphans", "--rmi", "local")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Logf("Podman Compose cleanup failed: %v: %s", err, output)
	}
}
