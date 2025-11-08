package operation_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/host"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		t.Run("builds images from compose file", func(t *testing.T) {
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
			build := operation.NewBuild(os.Stdout, composeFilePath, h)

			err := build.Run()

			require.NoError(t, err)
			testutil.RequireImageExists(t, h, imageName)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints build command", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			build := operation.NewBuild(os.Stdout, composeFilePath, host.Local)

			err := build.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf("docker compose -f %s build\n", composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}
