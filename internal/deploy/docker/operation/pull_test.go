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

func TestPull(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		t.Run("pulls images from compose file", func(t *testing.T) {
			h := host.Local
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  alpine:
    image: alpine:latest
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			pull := operation.NewPull(os.Stdout, composeFilePath, h)

			err := pull.Run()

			require.NoError(t, err)
			testutil.RequireImageExists(t, h, "alpine:latest")
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints pull command", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			pull := operation.NewPull(os.Stdout, composeFilePath, host.Local)

			err := pull.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf("docker compose -f %s pull\n", composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}
