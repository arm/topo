package operation_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDigestFromPushOutput(t *testing.T) {
	t.Run("parses digest from typical push output", func(t *testing.T) {
		output := `The push refers to repository [localhost:12737/myimage]
latest: digest: sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2 size: 1234`

		got, err := operation.ParseDigestFromPushOutput(output)

		require.NoError(t, err)
		assert.Equal(t, "sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", got)
	})

	t.Run("parses digest with surrounding output", func(t *testing.T) {
		output := `Using default tag: latest
The push refers to repository [localhost:12737/alpine]
5d3e392a13a0: Layer already exists
latest: digest: sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890 size: 528`

		got, err := operation.ParseDigestFromPushOutput(output)

		require.NoError(t, err)
		assert.Equal(t, "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", got)
	})

	t.Run("returns error when no digest found", func(t *testing.T) {
		output := `The push refers to repository [localhost:12737/myimage]
latest: size: 1234`

		_, err := operation.ParseDigestFromPushOutput(output)

		assert.EqualError(t, err, "no digest found in push output")
	})

	t.Run("returns error for empty output", func(t *testing.T) {
		_, err := operation.ParseDigestFromPushOutput("")

		assert.EqualError(t, err, "no digest found in push output")
	})
}

func TestRegistryTransfer(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		t.Run("it returns expected string", func(t *testing.T) {
			localHost := command.LocalHost
			transfer := operation.NewRegistryTransfer("any.yaml", localHost, localHost, operation.DefaultRegistryPort)

			got := transfer.Description()

			assert.Equal(t, "Transfer via registry", got)
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("it transfers images via registry", func(t *testing.T) {
			testutil.RequireLinuxDockerEngine(t)
			h := command.LocalHost
			port := operation.DefaultRegistryPort
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			dockerFilePath := filepath.Join(tmpDir, "Dockerfile")
			imageName := testutil.TestImageName(t)
			composeFileContent := fmt.Sprintf(`
services:
  test:
    build: .
    image: %s
`, imageName)
			dockerFileContent := `FROM alpine:latest`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			testutil.RequireWriteFile(t, dockerFilePath, dockerFileContent)

			buildCmd := command.DockerCompose(h, composeFilePath, "build")
			buildOut, err := buildCmd.CombinedOutput()
			require.NoError(t, err, "build failed: %s", string(buildOut))

			rmCmd := command.Docker(h, "rm", "-f", operation.RegistryContainerName)
			rmOut, rmErr := rmCmd.CombinedOutput()
			if rmErr != nil {
				t.Logf("registry container cleanup (expected if not running): %s", string(rmOut))
			}

			startReg := command.Docker(h, "run", "-d", "--restart=always", "-p", fmt.Sprintf("%s:5000", port), "--name", operation.RegistryContainerName, "registry:2")
			startOut, err := startReg.CombinedOutput()
			require.NoError(t, err, "could not start registry for test: %s", string(startOut))
			t.Cleanup(func() {
				rmReg := command.Docker(h, "rm", "-f", operation.RegistryContainerName)
				_ = rmReg.Run()
			})

			transfer := operation.NewRegistryTransfer(composeFilePath, h, h, port)
			err = transfer.Run(os.Stdout)
			require.NoError(t, err)
			testutil.RequireImageExists(t, h, imageName)
		})
	})
}
