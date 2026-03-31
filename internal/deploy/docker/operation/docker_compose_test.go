package operation_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
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
			op := operation.NewDockerCompose("", composeFilePath, command.LocalHost, []string{"config", "--services"})

			err := op.Run(&buf)

			require.NoError(t, err)
			assert.Contains(t, buf.String(), "test-service")
		})
	})
}

func TestNewDockerComposeBuild(t *testing.T) {
	op := operation.NewDockerComposeBuild("/path/to/compose.yaml", command.LocalHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Build images", got)
	})
}

func TestNewDockerComposePull(t *testing.T) {
	op := operation.NewDockerComposePull("/path/to/compose.yaml", command.LocalHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Pull images", got)
	})
}

func TestNewDockerComposeRun(t *testing.T) {
	opDefault := operation.NewDockerComposeUp("/path/to/compose.yaml", command.LocalHost, operation.RecreateModeDefault)

	t.Run("Description", func(t *testing.T) {
		got := opDefault.Description()

		assert.Equal(t, "Start services", got)
	})

	opForce := operation.NewDockerComposeUp("/path/to/compose.yaml", command.LocalHost, operation.RecreateModeForce)

	t.Run("Description with --force-recreate", func(t *testing.T) {
		got := opForce.Description()

		assert.Equal(t, "Start services", got)
	})
}
