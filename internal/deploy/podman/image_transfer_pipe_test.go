package podman_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/ssh"
	gtestutil "github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaPipe(t *testing.T) {
	requireLocalPodman(t)
	fixture := newImageTransferFixture(t)
	target := gtestutil.StartContainer(t, gtestutil.PodmanContainer)
	targetDestination := ssh.NewDestination(target.SSHDestination)
	remoteSocket, err := podman.NewRemoteSocket(context.Background(), targetDestination)
	require.NoError(t, err)
	err = podman.BuildImages(t.Context(), t.Output(), podman.LocalSocket, fixture.composeFile)
	require.NoError(t, err)

	err = podman.TransferImagesViaPipe(t.Context(), t.Output(), podman.LocalSocket, remoteSocket, fixture.composeFile)

	require.NoError(t, err)
	for _, imageName := range fixture.imageNames {
		assertImageExists(t, remoteSocket, imageName)
	}
}

func assertImageExists(t *testing.T, socket podman.Socket, imageName string) {
	t.Helper()
	err := podman.Command(t.Context(), socket, "image", "exists", imageName).Run()
	assert.NoError(t, err)
}
