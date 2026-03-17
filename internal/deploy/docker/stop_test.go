package docker_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeploymentStop(t *testing.T) {
	composeFile := "compose.yaml"

	t.Run("runs stop operation for remote host", func(t *testing.T) {
		remoteHost := ssh.Host("user@remote")

		got := docker.NewDeploymentStop(composeFile, remoteHost)

		want := goperation.Sequence{
			operation.NewDockerComposeStop(composeFile, remoteHost),
		}
		assert.Equal(t, want, got)
	})

	t.Run("runs stop operation for local host", func(t *testing.T) {
		got := docker.NewDeploymentStop(composeFile, ssh.PlainLocalhost)

		want := goperation.Sequence{
			operation.NewDockerComposeStop(composeFile, ssh.PlainLocalhost),
		}
		assert.Equal(t, want, got)
	})
}

func TestDeploymentStop(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints stop command", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  alpine:
    image: alpine:latest
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			targetHost := ssh.Host("user@remote")
			stop := docker.NewDeploymentStop(composeFilePath, targetHost)

			err := stop.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf(`
┌─ Stop services ───────────────────────────────────────
docker -H ssh://user@remote compose -f %[1]s stop
`, composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}
