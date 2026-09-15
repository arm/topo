package docker_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDeployment(t *testing.T) {
	requireDocker(t)

	t.Run("deploys to localhost", func(t *testing.T) {
		scope, imageName := deploymentFixture(t)
		t.Cleanup(func() { forceComposeDown(t, scope) })
		requireImageDoesNotExist(t, docker.LocalHost, imageName)
		deployOptions := docker.DeployOptions{TargetHost: ssh.PlainLocalhost}

		err := docker.Deploy(t.Context(), t.Output(), scope, deployOptions)

		require.NoError(t, err)
		requireImageExists(t, docker.LocalHost, imageName)
		assertContainersRunning(t, ssh.PlainLocalhost, scope)
	})

	t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
		container := startContainer(t, dinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		scope, imageName := deploymentFixture(t)
		requireImageDoesNotExist(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
		deployOptions := docker.DeployOptions{TargetHost: remoteDockerHost}

		err := docker.Deploy(t.Context(), t.Output(), scope, deployOptions)

		require.NoError(t, err)
		requireImageExists(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
		assertContainersRunning(t, remoteDockerHost, scope)
	})

	t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
		registryPort := requireAvailableTCPPort(t, "127.0.0.1")
		registryContainerName := testContainerName(t) + "-registry"
		cleanupRegistryContainer(t, registryContainerName)
		container := startContainer(t, dinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		remoteCommandHost := docker.NewHostFromDestination(remoteDockerHost)
		scope, imageName := deploymentFixture(t)
		requireImageDoesNotExist(t, remoteCommandHost, imageName)
		deployOptions := docker.DeployOptions{
			TargetHost: remoteDockerHost,
			Registry: &docker.RegistryConfig{
				ContainerName:       registryContainerName,
				Port:                registryPort,
				SkipRemotePortCheck: true,
			},
		}

		err := docker.Deploy(t.Context(), t.Output(), scope, deployOptions)

		require.NoError(t, err)
		requireImageExists(t, remoteCommandHost, imageName)
		assertContainersRunning(t, remoteDockerHost, scope)
	})
}

func deploymentFixture(t *testing.T) (project.Scope, string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	imageName := testImageName(t)
	composeFilePath := testutil.RequireWriteComposeFile(t, temporaryDirectory, fmt.Sprintf(`
name: %s
services:
  a-service:
    build: .
    image: %s
    stop_grace_period: 1s
`, testProjectName(t), imageName))
	testutil.RequireWriteFile(t, filepath.Join(temporaryDirectory, "Dockerfile"), `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		removeOutput, err := docker.Command(context.Background(), docker.LocalHost, "image", "rm", "-f", imageName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return project.Scope{ComposeFile: composeFilePath}, imageName
}

func cleanupRegistryContainer(t *testing.T, containerName string) {
	t.Helper()
	_ = docker.Command(t.Context(), docker.LocalHost, "rm", "-f", containerName).Run()
	t.Cleanup(func() {
		removeOutput, err := docker.Command(context.Background(), docker.LocalHost, "rm", "-f", containerName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove registry container: %v: %s", err, string(removeOutput))
		}
	})
}
