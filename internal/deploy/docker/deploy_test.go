package docker_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/host"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployment(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("builds images, transfers them, and starts services", func(t *testing.T) {
		// Note: This test doesn't perfectly verify that the image was transferred through
		// the pipe rather than just existing on the target.
		// To properly test this, we would need to ensure that test has access to two docker engines,
		// which could be achieved with dind, limavm, etc.
		// As a temporary compromise, this test verifies the operations complete without error
		// and the containers are running requested images afterward.
		tmpDir := t.TempDir()
		composeFilePath := filepath.Join(tmpDir, "compose.yaml")
		composeFileContent := fmt.Sprintf(`
name: %s
services:
  alpine:
    image: alpine:latest
    command: tail -f /dev/null
    restart: unless-stopped
`, testutil.TestProjectName(t))
		testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
		t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })
		d := docker.NewDeployment(os.Stdout, composeFilePath, host.Local)

		err := d.Run()

		require.NoError(t, err)
		testutil.AssertContainersRunning(t, composeFilePath)
	})

	t.Run("dry run prints commands without executing", func(t *testing.T) {
		var buf bytes.Buffer
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
		targetHost := host.NewSSH("ssh://user@remote")
		d := docker.NewDeployment(&buf, composeFilePath, targetHost)

		err := d.DryRun(&buf)

		require.NoError(t, err)
		got := buf.String()
		want := fmt.Sprintf(`docker compose -f %[1]s build
docker compose -f %[1]s pull
docker save alpine:latest | docker -H ssh://user@remote load
docker save nginx:latest | docker -H ssh://user@remote load
docker -H ssh://user@remote compose -f %[1]s up -d --no-build --pull never
`, composeFilePath)
		assert.Equal(t, want, got)
	})
}
