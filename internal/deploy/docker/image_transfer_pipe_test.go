package docker_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaPipe(t *testing.T) {
	requireLinuxDockerEngine(t)
	sourceHost := docker.LocalHost
	scope, imageName := buildTransferTestImage(t, sourceHost)

	destinationContainer := startContainer(t, dinDContainer)
	destination := ssh.NewDestination(destinationContainer.SSHDestination)
	destinationHost := docker.NewHostFromDestination(destination)
	requireImageDoesNotExist(t, destinationHost, imageName)

	err := docker.TransferImagesViaPipe(t.Context(), os.Stdout, sourceHost, destinationHost, scope)

	require.NoError(t, err)
	requireImageExists(t, destinationHost, imageName)
}

func buildTransferTestImage(t *testing.T, host docker.Host) (project.Scope, string) {
	t.Helper()
	temporaryDirectory := t.TempDir()
	dockerFilePath := filepath.Join(temporaryDirectory, "Dockerfile")
	imageName := testImageName(t)
	composeFilePath := testutil.RequireWriteComposeFile(t, temporaryDirectory, fmt.Sprintf(`
services:
  test:
    build: .
    image: %s
`, imageName))
	testutil.RequireWriteFile(t, dockerFilePath, "FROM alpine:latest")

	scope := project.Scope{ComposeFile: composeFilePath}
	buildCommand := docker.ComposeCommand(t.Context(), host, scope, "build")
	buildOutput, err := buildCommand.CombinedOutput()
	require.NoError(t, err, "failed to build image: %s", string(buildOutput))
	return scope, imageName
}
