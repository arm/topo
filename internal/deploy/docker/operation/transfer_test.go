package operation_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/command"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	"github.com/arm-debug/topo-cli/internal/deploy/host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransfer(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Description", func(t *testing.T) {
		h := host.Local
		tmpDir := t.TempDir()
		composeFilePath := filepath.Join(tmpDir, "compose.yaml")
		transfer := operation.NewTransfer(os.Stdout, composeFilePath, h, h)

		got := transfer.Description()

		assert.Equal(t, "Transfer images", got)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("transfers images from source to target", func(t *testing.T) {
			// Note: The Run test doesn't perfectly verify that the image was transferred through
			// the pipe rather than just existing on the target.
			// To properly test this, we would need to either:
			// - Remove the image after save but before load (not feasible with current implementation).
			// - Ensure test has access to two docker engines (expensive).
			// As a compromise, this test verifies the operation completes without error and the image exists afterward.
			h := host.Local
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

			transfer := operation.NewTransfer(os.Stdout, composeFilePath, h, h)

			err = transfer.Run()

			require.NoError(t, err)
			testutil.RequireImageExists(t, h, imageName)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints transfer commands", func(t *testing.T) {
			var buf bytes.Buffer
			h := host.Local
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  alpine:
    image: alpine:latest
  nginx:
    image: nginx:latest
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			transfer := operation.NewTransfer(os.Stdout, composeFilePath, h, host.New("ssh://user@remote"))

			err := transfer.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := `docker save alpine:latest | docker -H ssh://user@remote load
docker save nginx:latest | docker -H ssh://user@remote load
`
			assert.Equal(t, want, got)
		})
	})
}
