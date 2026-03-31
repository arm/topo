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

func TestDockerComposePipeTransfer(t *testing.T) {
	t.Run("Description", func(t *testing.T) {
		h := command.LocalHost
		tmpDir := t.TempDir()
		composeFilePath := filepath.Join(tmpDir, "compose.yaml")
		transfer := operation.NewDockerComposePipeTransfer(composeFilePath, h, h)

		got := transfer.Description()

		assert.Equal(t, "Transfer images", got)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("transfers images from source to target", func(t *testing.T) {
			testutil.RequireLinuxDockerEngine(t)
			// Note: The Run test doesn't perfectly verify that the image was transferred through
			// the pipe rather than just existing on the target.
			// To properly test this, we would need to either:
			// - Remove the image after save but before load (not feasible with current implementation).
			// - Ensure test has access to two docker engines (expensive).
			// As a compromise, this test verifies the operation completes without error and the image exists afterward.
			h := command.LocalHost
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
			buildOutput, err := buildCmd.CombinedOutput()
			require.NoError(t, err, "failed to build image: %s", string(buildOutput))

			transfer := operation.NewDockerComposePipeTransfer(composeFilePath, h, h)

			err = transfer.Run(os.Stdout)

			require.NoError(t, err)
			testutil.RequireImageExists(t, h, imageName)
		})
	})
}
