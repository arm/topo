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

type imageTransferFixture struct {
	composeFile string
	imageNames  []string
}

func newImageTransferFixture(t *testing.T) imageTransferFixture {
	t.Helper()

	dir := t.TempDir()
	testName := sanitiseTestName(t)
	projectName := "test-project-" + testName
	explicitImageName := "test-image-" + testName
	generatedImageName := projectName + "_generated"
	composeFile := filepath.Join(dir, "compose.yaml")

	requireWriteFile(t, composeFile, fmt.Sprintf(`
name: %s
services:
  explicit:
    build: .
    image: %s
  generated:
    build: .
`, projectName, explicitImageName))
	requireWriteFile(t, filepath.Join(dir, "Dockerfile"), `FROM docker.io/library/alpine:latest`)

	fixture := imageTransferFixture{
		composeFile: composeFile,
		imageNames:  []string{explicitImageName, generatedImageName},
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, imageName := range fixture.imageNames {
			_ = podman.Command(ctx, podman.LocalSocket, "image", "rm", "-f", imageName).Run()
		}
	})
	return fixture
}
