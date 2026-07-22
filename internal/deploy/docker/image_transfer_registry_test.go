package docker_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferImagesViaRegistry(t *testing.T) {
	testutil.RequireLinuxDockerEngine(t)
	host := command.LocalHost
	const (
		registryContainerName = "topo-test-registry"
		registryPort          = "12738"
	)
	composeFilePath, imageName := buildTransferTestImage(t, host)

	removeRegistry := command.Docker(host, "rm", "-f", registryContainerName)
	removeOutput, removeErr := removeRegistry.CombinedOutput()
	if removeErr != nil {
		t.Logf("registry container cleanup (expected if not running): %s", string(removeOutput))
	}

	startRegistry := command.Docker(host, "run", "-d", "--restart=always", "-p", fmt.Sprintf("%s:5000", registryPort), "--name", registryContainerName, "registry:2")
	startOutput, err := startRegistry.CombinedOutput()
	require.NoError(t, err, "could not start registry for test: %s", string(startOutput))
	t.Cleanup(func() {
		_ = command.Docker(host, "rm", "-f", registryContainerName).Run()
	})

	destinationContainer := testutil.StartContainer(t, testutil.DinDContainer)
	destination := ssh.NewDestination(destinationContainer.SSHDestination)
	destinationHost := command.NewHostFromDestination(destination)
	testutil.RequireImageDoesNotExist(t, destinationHost, imageName)

	tunnel, err := ssh.OpenSSHTunnel(context.Background(), os.Stdout, destination, registryPort, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, tunnel.Close(context.Background(), os.Stdout))
	})

	err = docker.TransferImagesViaRegistry(t.Context(), os.Stdout, composeFilePath, host, destinationHost, registryPort)

	require.NoError(t, err)
	testutil.RequireImageExists(t, destinationHost, imageName)
}

func TestParseDigestFromPushOutput(t *testing.T) {
	t.Run("parses digest from typical push output", func(t *testing.T) {
		output := `The push refers to repository [localhost:12737/myimage]
latest: digest: sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 size: 1234`

		got, err := docker.ParseDigestFromPushOutput(output)

		require.NoError(t, err)
		assert.Equal(t, "sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", got)
	})

	t.Run("parses digest with surrounding output", func(t *testing.T) {
		output := `Using default tag: latest
The push refers to repository [localhost:12737/alpine]
5d3e392a13a0: Layer already exists
latest: digest: sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890 size: 528`

		got, err := docker.ParseDigestFromPushOutput(output)

		require.NoError(t, err)
		assert.Equal(t, "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", got)
	})

	t.Run("returns error when no digest found", func(t *testing.T) {
		output := `The push refers to repository [localhost:12737/myimage]
latest: size: 1234`

		_, err := docker.ParseDigestFromPushOutput(output)

		assert.EqualError(t, err, "no digest found in push output")
	})

	t.Run("returns error for empty output", func(t *testing.T) {
		_, err := docker.ParseDigestFromPushOutput("")

		assert.EqualError(t, err, "no digest found in push output")
	})
}
