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
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	requireLocalPodman(t)
	composeFile := deploymentFixture(t)
	t.Cleanup(func() { cleanupComposeProject(t, composeFile) })

	err := podman.Deploy(t.Context(), io.Discard, composeFile)

	require.NoError(t, err)
	assertContainersRunning(t, composeFile)
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

func assertContainersRunning(t *testing.T, composeFile string) {
	t.Helper()
	cmd := podman.ComposeCommand(t.Context(), composeFile, "ps", "--format", "json", "--all")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	require.NotEmpty(t, bytes.TrimSpace(output), "no containers reported")

	containers, err := unmarshalNDJSON(output)
	require.NoError(t, err)

	for _, container := range containers {
		assert.Equal(t, "running", container["State"], "container %s is not running: %s", container["Name"], container["State"])
	}
}

type jsonObject map[string]any

func unmarshalNDJSON(ndJSON []byte) ([]jsonObject, error) {
	objects := []jsonObject{}
	lines := bytes.SplitSeq(bytes.TrimSpace(ndJSON), []byte("\n"))

	for line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var object jsonObject
		if err := json.Unmarshal(line, &object); err != nil {
			return objects, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func deploymentFixture(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	testName := gtestutil.SanitiseTestName(t)
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
	gtestutil.RequireWriteFile(t, composeFile, composeFileContent)
	gtestutil.RequireWriteFile(t, filepath.Join(tempDir, "Dockerfile"), `
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
	return composeFile
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
