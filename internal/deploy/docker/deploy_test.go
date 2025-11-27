package docker_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	"github.com/arm-debug/topo-cli/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployment(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("Run", func(t *testing.T) {
		dockerVM := testutil.StartDockerVM(t)

		t.Run("builds images, transfers them, and starts services", func(t *testing.T) {
			remoteDockerHost := ssh.Host(dockerVM.SSHConnectionString)
			tmpDir := t.TempDir()
			dockerFilePath := filepath.Join(tmpDir, "Dockerfile")
			dockerFileContent := `
FROM alpine:latest
CMD ["tail", "-f", "/dev/null"]
`
			testutil.RequireWriteFile(t, dockerFilePath, dockerFileContent)
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := fmt.Sprintf(`
name: %s
services:
  busybox:
    image: busybox
    command: ["tail", "-f", "/dev/null"]
  a-service:
    build: .
`, testutil.TestProjectName(t))
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			t.Cleanup(func() { testutil.ForceComposeDown(t, composeFilePath) })
			d := docker.NewDeployment(composeFilePath, remoteDockerHost)

			err := d.Run(os.Stdout)

			require.NoError(t, err)
			testutil.AssertContainersRunning(t, remoteDockerHost, composeFilePath)
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints all commands", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  alpine:
    image: alpine:latest
  busybox:
    image: busybox
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			targetHost := ssh.Host("user@remote")
			d := docker.NewDeployment(composeFilePath, targetHost)

			err := d.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf(`
┌─ Build images ────────────────────────────────────────
docker compose -f %[1]s build

┌─ Pull images ─────────────────────────────────────────
docker compose -f %[1]s pull

┌─ Transfer images ─────────────────────────────────────
docker save alpine:latest | docker -H ssh://user@remote load
docker save busybox | docker -H ssh://user@remote load

┌─ Start services ──────────────────────────────────────
docker -H ssh://user@remote compose -f %[1]s up -d --no-build --pull never
`, composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}
