package podman_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/podman"
	gtestutil "github.com/arm/topo/internal/testutil"
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
