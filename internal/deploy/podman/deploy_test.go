package podman_test

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
	t.Run("rejects runtime before accessing Podman", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  firmware:
    image: alpine
    runtime: io.containerd.remoteproc.v1
`)

		err := podman.Deploy(t.Context(), &bytes.Buffer{}, project.Scope{ComposeFile: composeFile}, deploy.Options{})

		require.ErrorContains(t, err, `specifying "runtime:" in Compose files is unsupported for Podman deployments`)
	})

	t.Run("deploys to localhost", func(t *testing.T) {
		testutil.RequirePodman(t)
		scope, projectName := deploymentFixture(t)
		t.Cleanup(func() { cleanupComposeProject(t, scope) })
		options := deploy.Options{TargetHost: ssh.PlainLocalhost}

		err := podman.Deploy(t.Context(), t.Output(), scope, options)

		require.NoError(t, err)
		assertContainersRunning(t, projectName, podman.LocalSocket)
	})

	t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
		testutil.RequirePodman(t)
		podmanContainer := startPodmanInContainer(t)
		scope, projectName := deploymentFixture(t)
		targetDestination := ssh.NewDestination(podmanContainer.SSHDestination)
		options := deploy.Options{TargetHost: targetDestination}

		err := podman.Deploy(t.Context(), t.Output(), scope, options)

		require.NoError(t, err)
		tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tunnel.Close())
		})
		assertContainersRunning(t, projectName, podman.NewSocket(tunnel.SocketURL()))
	})

	t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
		testutil.RequirePodman(t)
		registryContainerName := "topo-test-registry-" + sanitiseTestName(t)
		registryPort := startTestRegistry(t, registryContainerName)
		podmanContainer := startPodmanInContainer(t)
		scope, projectName := deploymentFixture(t)
		targetDestination := ssh.NewDestination(podmanContainer.SSHDestination)
		options := deploy.Options{
			TargetHost: targetDestination,
			Registry: &deploy.RegistryConfig{
				ContainerName: registryContainerName,
				Port:          registryPort,
			},
		}

		err := podman.Deploy(t.Context(), t.Output(), scope, options)

		require.NoError(t, err)
		tunnel, err := podman.TunnelRemoteSocketPath(context.Background(), t.Output(), targetDestination)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, tunnel.Close())
		})
		assertContainersRunning(t, projectName, podman.NewSocket(tunnel.SocketURL()))
	})
}

func deploymentFixture(t *testing.T) (project.Scope, string) {
	t.Helper()
	tempDir := t.TempDir()
	testName := sanitiseTestName(t)
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
	composeFileContent, err := testutil.FixPodmanInDockerQuirk(composeFileContent)
	require.NoError(t, err)
	composeFile := testutil.RequireWriteComposeFile(t, tempDir, composeFileContent)
	testutil.RequireWriteFile(t, filepath.Join(tempDir, "Dockerfile"), `
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

func cleanupComposeProject(t *testing.T, scope project.Scope) {
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
