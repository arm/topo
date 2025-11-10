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

func TestRun(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		t.Run("starts services from compose file", func(t *testing.T) {
			imageName := testutil.TestImageName(t)
			testutil.BuildMinimalImage(t, host.Local, imageName)
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := fmt.Sprintf(`
name: %s
services:
  a-service:
    image: %s
`, testutil.TestProjectName(t), imageName)
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })
			run := operation.NewRun(os.Stdout, composeFilePath, host.Local)

			err := run.Run()

			require.NoError(t, err)
			testutil.AssertContainersRunning(t, composeFilePath)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints run command", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			run := operation.NewRun(os.Stdout, composeFilePath, host.Local)

			err := run.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf("docker compose -f %s up -d --no-build --pull never\n", composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}
