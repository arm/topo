package podman_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaRegistry(t *testing.T) {
	requireLocalPodman(t)
	registryContainerName := "topo-test-registry-transfer-" + sanitiseTestName(t)
	requireRegistryContainerAbsent(t, registryContainerName)
	registryPort := requireAvailableTCPPort(t)
	var output bytes.Buffer
	require.NoError(t, podman.EnsureRegistryRunning(t.Context(), &output, registryContainerName, registryPort), output.String())

	composeFile, imageName := imageTransferFixture(t)
	require.NoError(t, podman.BuildImages(t.Context(), t.Output(), podman.LocalSocket, composeFile))
	target := startContainer(t)
	targetDestination := ssh.NewDestination(target.SSHDestination)
	targetSocketTunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, targetSocketTunnel.Close())
	})
	targetSocket := podman.NewSocket(targetSocketTunnel.SocketURL())
	assertImageDoesNotExist(t, targetSocket, imageName)

	registryTunnel, err := ssh.OpenTunnel(context.Background(), t.Output(), targetDestination, registryPort)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, registryTunnel.Close(context.Background(), t.Output()))
	})

	err = podman.TransferImagesViaRegistry(t.Context(), t.Output(), podman.LocalSocket, targetSocket, composeFile, registryPort)

	require.NoError(t, err)
	assertImageExists(t, targetSocket, imageName)
}

func assertImageDoesNotExist(t *testing.T, socket podman.Socket, imageName string) {
	t.Helper()
	err := podman.Command(t.Context(), socket, "image", "exists", imageName).Run()
	assert.Error(t, err)
}
