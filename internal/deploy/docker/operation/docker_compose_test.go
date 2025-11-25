package operation_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/deploy/docker/operation"
	"github.com/arm-debug/topo-cli/internal/deploy/docker/testutil"
	"github.com/arm-debug/topo-cli/internal/deploy/host"
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
			op := operation.NewDockerCompose("", &buf, composeFilePath, host.Local, []string{"config", "--services"})

			err := op.Run()

			require.NoError(t, err)
			assert.Contains(t, buf.String(), "test-service")
		})
	})

	t.Run("DryRun", func(t *testing.T) {
		t.Run("prints command with multiple args and remote host", func(t *testing.T) {
			var buf bytes.Buffer
			tmpDir := t.TempDir()
			composeFilePath := filepath.Join(tmpDir, "compose.yaml")
			remoteHost := host.New("ssh://user@remote")
			op := operation.NewDockerCompose("", &buf, composeFilePath, remoteHost, []string{"up", "-d", "--no-build"})

			err := op.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf("docker -H ssh://user@remote compose -f %s up -d --no-build\n", composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}

func TestNewBuild(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := host.New("ssh://user@remote")
	op := operation.NewBuild(nil, composeFilePath, remoteHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Build images", got)
	})

	t.Run("DryRun", func(t *testing.T) {
		var buf bytes.Buffer

		err := op.DryRun(&buf)

		require.NoError(t, err)
		want := fmt.Sprintf("docker -H %s compose -f %s build\n", remoteHost, composeFilePath)
		assert.Equal(t, want, buf.String())
	})
}

func TestNewPull(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := host.New("ssh://user@remote")
	op := operation.NewPull(nil, composeFilePath, remoteHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Pull images", got)
	})

	t.Run("DryRun", func(t *testing.T) {
		var buf bytes.Buffer

		err := op.DryRun(&buf)

		require.NoError(t, err)
		want := fmt.Sprintf("docker -H %s compose -f %s pull\n", remoteHost, composeFilePath)
		assert.Equal(t, want, buf.String())
	})
}

func TestNewRun(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := host.New("ssh://user@remote")
	op := operation.NewRun(nil, composeFilePath, remoteHost)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Start services", got)
	})

	t.Run("DryRun", func(t *testing.T) {
		var buf bytes.Buffer

		err := op.DryRun(&buf)

		require.NoError(t, err)
		want := fmt.Sprintf("docker -H %s compose -f %s up -d --no-build --pull never\n", remoteHost, composeFilePath)
		assert.Equal(t, want, buf.String())
	})
}
