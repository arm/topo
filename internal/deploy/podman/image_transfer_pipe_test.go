package podman_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaPipe(t *testing.T) {
	requireLocalPodman(t)
	composeFile, imageName := imageTransferFixture(t)
	target := gtestutil.StartContainer(t, gtestutil.PodmanContainer)
	targetHost := ssh.NewDestination(target.SSHDestination)
	tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetHost)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, tunnel.Close())
	})
	remoteSocket := podman.NewSocket(tunnel.SocketURL())
	err = podman.BuildImages(t.Context(), t.Output(), podman.LocalSocket, composeFile)
	require.NoError(t, err)

	err = podman.TransferImagesViaPipe(t.Context(), t.Output(), podman.LocalSocket, remoteSocket, composeFile)

	require.NoError(t, err)
	assertImageExists(t, remoteSocket, imageName)
}

func assertImageExists(t *testing.T, socket podman.Socket, imageName string) {
	t.Helper()
	err := podman.Command(t.Context(), socket, "image", "exists", imageName).Run()
	assert.NoError(t, err)
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
