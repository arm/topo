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
			op := operation.NewDockerCompose(&buf, composeFilePath, host.Local, []string{"config", "--services"})

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
			op := operation.NewDockerCompose(&buf, composeFilePath, remoteHost, []string{"up", "-d", "--no-build"})

			err := op.DryRun(&buf)

			require.NoError(t, err)
			got := buf.String()
			want := fmt.Sprintf("docker -H ssh://user@remote compose -f %s up -d --no-build\n", composeFilePath)
			assert.Equal(t, want, got)
		})
	})
}

func TestOperationFactories(t *testing.T) {
	composeFilePath := "/path/to/compose.yaml"
	remoteHost := host.New("ssh://user@remote")

	tests := []struct {
		name        string
		op          *operation.DockerCompose
		wantCommand string
	}{
		{
			name:        "NewBuild",
			op:          operation.NewBuild(nil, composeFilePath, remoteHost),
			wantCommand: fmt.Sprintf("docker -H %s compose -f %s build\n", remoteHost, composeFilePath),
		},
		{
			name:        "NewPull",
			op:          operation.NewPull(nil, composeFilePath, remoteHost),
			wantCommand: fmt.Sprintf("docker -H %s compose -f %s pull\n", remoteHost, composeFilePath),
		},
		{
			name:        "NewRun",
			op:          operation.NewRun(nil, composeFilePath, remoteHost),
			wantCommand: fmt.Sprintf("docker -H %s compose -f %s up -d --no-build --pull never\n", remoteHost, composeFilePath),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			err := tt.op.DryRun(&buf)

			require.NoError(t, err)
			assert.Equal(t, tt.wantCommand, buf.String())
		})
	}
}
