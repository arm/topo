package operation_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerCompose(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		testutil.RequireDocker(t)

		t.Run("executes docker compose command with compose file", func(t *testing.T) {
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			composeFileContent := `
services:
  test-service:
    image: alpine:latest
`
			testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
			var buf bytes.Buffer
			op := operation.NewDockerCompose("", composeFilePath, ssh.PlainLocalhost, []string{"config", "--services"})

			err := op.Run(&buf)

			require.NoError(t, err)
			assert.Contains(t, buf.String(), "test-service")
		})
	})
}

func TestNewDockerComposeBuild(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := ssh.NewDestination("user@remote")
	op := operation.NewDockerComposeBuild(composeFilePath, remoteHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Build images", got)
	})
}

func TestNewDockerComposePull(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := ssh.NewDestination("user@remote")
	op := operation.NewDockerComposePull(composeFilePath, remoteHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Pull images", got)
	})
}

func TestNewDockerComposeRun(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := ssh.NewDestination("user@remote")
	opDefault := operation.NewDockerComposeUp(composeFilePath, remoteHost, operation.RecreateModeDefault)

	t.Run("Description", func(t *testing.T) {
		got := opDefault.Description()

		assert.Equal(t, "Start services", got)
	})

	opForce := operation.NewDockerComposeUp(composeFilePath, remoteHost, operation.RecreateModeForce)

	t.Run("Description with --force-recreate", func(t *testing.T) {
		got := opForce.Description()

		assert.Equal(t, "Start services", got)
	})
}
