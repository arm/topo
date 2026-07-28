package docker_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaPipe(t *testing.T) {
	requireLinuxDockerEngine(t)
	sourceHost := docker.LocalHost
	composeFilePath, imageName := buildTransferTestImage(t, sourceHost)

	destinationContainer := startContainer(t, dinDContainer)
	destination := ssh.NewDestination(destinationContainer.SSHDestination)
	destinationHost := docker.NewHostFromDestination(destination)
	requireImageDoesNotExist(t, destinationHost, imageName)

	err := docker.TransferImagesViaPipe(t.Context(), os.Stdout, sourceHost, destinationHost, composeFilePath)

	require.NoError(t, err)
	requireImageExists(t, destinationHost, imageName)
}

func buildTransferTestImage(t *testing.T, host docker.Host) (string, string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	composeFilePath := filepath.Join(temporaryDirectory, "compose.yaml")
	dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
	imageName := testImageName(t)
	composeFileContent := fmt.Sprintf(`
services:
  test:
    build: .
    image: %s
`, imageName)
	requireWriteFile(t, composeFilePath, composeFileContent)
	requireWriteFile(t, dockerFilePath, "FROM alpine:latest")

	buildCommand := docker.ComposeCommand(t.Context(), host, composeFilePath, "build")
	buildOutput, err := buildCommand.CombinedOutput()
	require.NoError(t, err, "failed to build image: %s", string(buildOutput))
	return composeFilePath, imageName
}
