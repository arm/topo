package docker_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaPipe(t *testing.T) {
	testutil.RequireLinuxDockerEngine(t)
	sourceHost := command.LocalHost
	composeFilePath, imageName := buildTransferTestImage(t, sourceHost)

	destinationContainer := testutil.StartContainer(t, testutil.DinDContainer)
	destination := ssh.NewDestination(destinationContainer.SSHDestination)
	destinationHost := command.NewHostFromDestination(destination)
	testutil.RequireImageDoesNotExist(t, destinationHost, imageName)

	err := docker.TransferImagesViaPipe(t.Context(), os.Stdout, composeFilePath, sourceHost, destinationHost)

	require.NoError(t, err)
	testutil.RequireImageExists(t, destinationHost, imageName)
}

func buildTransferTestImage(t *testing.T, host command.Host) (string, string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	composeFilePath := filepath.Join(temporaryDirectory, "compose.yaml")
	dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
	imageName := testutil.TestImageName(t)
	composeFileContent := fmt.Sprintf(`
services:
  test:
    build: .
    image: %s
`, imageName)
	testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
	testutil.RequireWriteFile(t, dockerFilePath, "FROM alpine:latest")

	buildCommand := command.DockerCompose(host, composeFilePath, "build")
	buildOutput, err := buildCommand.CombinedOutput()
	require.NoError(t, err, "failed to build image: %s", string(buildOutput))
	return composeFilePath, imageName
}
