package podman_test

import (
	"context"
	"io"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestStop(t *testing.T) {
	requireLocalPodman(t)

	t.Run("stops services on localhost", func(t *testing.T) {
		composeFile, projectName := deploymentFixture(t)
		t.Cleanup(func() { cleanupComposeProject(t, composeFile) })
		options := podman.DeployOptions{TargetHost: ssh.PlainLocalhost}
		require.NoError(t, podman.Deploy(t.Context(), t.Output(), composeFile, options))

		err := podman.Stop(t.Context(), t.Output(), composeFile, ssh.PlainLocalhost)

		require.NoError(t, err)
		assertContainersStopped(t, projectName, podman.LocalSocket)
	})

	t.Run("stops services on a remote target", func(t *testing.T) {
		podmanContainer := startPodmanInContainer(t)
		composeFile, projectName := deploymentFixture(t)
		target := ssh.NewDestination(podmanContainer.SSHDestination)
		require.NoError(t, podman.Deploy(t.Context(), t.Output(), composeFile, podman.DeployOptions{TargetHost: target}))

		err := podman.Stop(t.Context(), t.Output(), composeFile, target)

		require.NoError(t, err)
		tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), io.Discard, target)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tunnel.Close()) })
		assertContainersStopped(t, projectName, podman.NewSocket(tunnel.SocketURL()))
	})
}
