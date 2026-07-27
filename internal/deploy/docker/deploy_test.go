package docker_test

import (
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestDeployment(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("deploys to localhost", func(t *testing.T) {
		composeFilePath, imageName := deploymentFixture(t)
		t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })
		testutil.RequireImageDoesNotExist(t, docker.LocalHost, imageName)
		deployOptions := docker.DeployOptions{TargetHost: ssh.PlainLocalhost}

		err := docker.Deploy(t.Context(), io.Discard, composeFilePath, deployOptions)

		require.NoError(t, err)
		testutil.RequireImageExists(t, docker.LocalHost, imageName)
		testutil.AssertContainersRunning(t, ssh.PlainLocalhost, composeFilePath)
	})

	t.Run("transfers images to a remote host via pipe", func(t *testing.T) {
		container := testutil.StartContainer(t, testutil.DinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		composeFilePath, imageName := deploymentFixture(t)
		testutil.RequireImageDoesNotExist(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
		deployOptions := docker.DeployOptions{TargetHost: remoteDockerHost}

		err := docker.Deploy(t.Context(), io.Discard, composeFilePath, deployOptions)

		require.NoError(t, err)
		testutil.RequireImageExists(t, docker.NewHostFromDestination(remoteDockerHost), imageName)
		testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)
	})

	t.Run("transfers images to a remote host through a registry", func(t *testing.T) {
		registryPort := testutil.RequireAvailableTCPPort(t, "127.0.0.1")
		registryContainerName := testutil.TestContainerName(t) + "-registry"
		cleanupRegistryContainer(t, registryContainerName)
		container := testutil.StartContainer(t, testutil.DinDContainer)
		remoteDockerHost := ssh.NewDestination(container.SSHDestination)
		remoteCommandHost := docker.NewHostFromDestination(remoteDockerHost)
		composeFilePath, imageName := deploymentFixture(t)
		testutil.RequireImageDoesNotExist(t, remoteCommandHost, imageName)
		deployOptions := docker.DeployOptions{
			TargetHost: remoteDockerHost,
			Registry: &docker.RegistryConfig{
				ContainerName:       registryContainerName,
				Port:                registryPort,
				SkipRemotePortCheck: true,
			},
		}

		err := docker.Deploy(t.Context(), io.Discard, composeFilePath, deployOptions)

		require.NoError(t, err)
		testutil.RequireImageExists(t, remoteCommandHost, imageName)
		testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)
	})
}

func deploymentFixture(t *testing.T) (composeFilePath, imageName string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	imageName = testutil.TestImageName(t)
	composeFilePath = filepath.Join(temporaryDirectory, "compose.yaml")
	composeFileContent := fmt.Sprintf(`
name: %s
services:
  a-service:
    build: .
    image: %s
`, testutil.TestProjectName(t), imageName)
	testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
	testutil.RequireWriteFile(t, filepath.Join(temporaryDirectory, "Dockerfile"), `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`)
	t.Cleanup(func() {
		removeOutput, err := docker.Docker(docker.LocalHost, "image", "rm", "-f", imageName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove image %s: %v: %s", imageName, err, string(removeOutput))
		}
	})
	return composeFilePath, imageName
}

func cleanupRegistryContainer(t *testing.T, containerName string) {
	t.Helper()
	_ = docker.Docker(docker.LocalHost, "rm", "-f", containerName).Run()
	t.Cleanup(func() {
		removeOutput, err := docker.Docker(docker.LocalHost, "rm", "-f", containerName).CombinedOutput()
		if err != nil {
			t.Logf("failed to remove registry container: %v: %s", err, string(removeOutput))
		}
	})
}
