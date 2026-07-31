package docker_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestDeployment(t *testing.T) {
	requireDocker(t)

	t.Run("deploys to localhost", func(t *testing.T) {
		composeFilePath, imageName := deploymentFixture(t)
		t.Cleanup(func() { forceComposeDown(t, composeFilePath) })
		requireImageDoesNotExist(t, docker.LocalHost, imageName)
		deployOptions := docker.DeployOptions{TargetHost: ssh.PlainLocalhost}

		err := docker.Deploy(t.Context(), t.Output(), composeFilePath, deployOptions)

		require.NoError(t, err)
		requireImageExists(t, docker.LocalHost, imageName)
		assertContainersRunning(t, ssh.PlainLocalhost, composeFilePath)
	})

	t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
		container := startContainer(t, dinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		composeFilePath, imageName := deploymentFixture(t)
		requireImageDoesNotExist(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
		deployOptions := docker.DeployOptions{TargetHost: remoteDockerHost}

		err := docker.Deploy(t.Context(), t.Output(), composeFilePath, deployOptions)

		require.NoError(t, err)
		requireImageExists(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
		assertContainersRunning(t, remoteDockerHost, composeFilePath)
	})

	t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
		registryPort := requireAvailableTCPPort(t, "127.0.0.1")
		registryContainerName := testContainerName(t) + "-registry"
		cleanupRegistryContainer(t, registryContainerName)
		container := startContainer(t, dinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		remoteCommandHost := docker.NewHostFromDestination(remoteDockerHost)
		composeFilePath, imageName := deploymentFixture(t)
		requireImageDoesNotExist(t, remoteCommandHost, imageName)
		deployOptions := docker.DeployOptions{
			TargetHost: remoteDockerHost,
			Registry: &docker.RegistryConfig{
				ContainerName:       registryContainerName,
				Port:                registryPort,
				SkipRemotePortCheck: true,
			},
		}

		err := docker.Deploy(t.Context(), t.Output(), composeFilePath, deployOptions)

		require.NoError(t, err)
		requireImageExists(t, remoteCommandHost, imageName)
		assertContainersRunning(t, remoteDockerHost, composeFilePath)
	})
}

func deploymentFixture(t *testing.T) (composeFilePath, imageName string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	imageName = testImageName(t)
	composeFilePath = filepath.Join(temporaryDirectory, "compose.yaml")
	composeFileContent := fmt.Sprintf(`
name: %s
services:
  a-service:
    build: .
    image: %s
`, testProjectName(t), imageName)
	requireWriteFile(t, composeFilePath, composeFileContent)
	requireWriteFile(t, filepath.Join(temporaryDirectory, "Dockerfile"), `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		removeOutput, err := docker.Command(context.Background(), docker.LocalHost, "image", "rm", "-f", imageName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return composeFilePath, imageName
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
