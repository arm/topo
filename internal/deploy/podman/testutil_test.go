package podman_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

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

func requireWriteFile(t *testing.T, path, content string) {
	t.Helper()
	gtestutil.RequireWriteFile(t, path, content)
}

func requireAvailableTCPPort(t *testing.T) string {
	t.Helper()
	return gtestutil.RequireAvailableTCPPort(t, "127.0.0.1")
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

func imageTransferFixture(t *testing.T) (string, string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	composeFile := filepath.Join(temporaryDirectory, "compose.yaml")
	imageName := "test-image-" + sanitiseTestName(t)
	requireWriteFile(t, composeFile, fmt.Sprintf(`
services:
  test:
    build: .
    image: %s
`, imageName))
	requireWriteFile(t, filepath.Join(temporaryDirectory, "Dockerfile"), "FROM docker.io/library/alpine:latest\n")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = podman.Command(ctx, podman.LocalSocket, "image", "rm", "-f", imageName).Run()
	})
	return composeFile, imageName
}
