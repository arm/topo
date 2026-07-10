package post_deploy_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploySuccess(t *testing.T) {
	t.Run("Run writes deployment_success_message from compose file", func(t *testing.T) {
		dir := t.TempDir()
		composeFile := filepath.Join(dir, "compose.yaml")
		testutil.RequireWriteFile(t, composeFile, `
x-topo:
  deployment_success_message: "Deployment complete!"
services:
  app:
    image: nginx
`)
		op := post_deploy.NewDeploySuccess(composeFile, command.LocalHost, "Run `topo ps` to see deployed containers")
		var buf bytes.Buffer

		err := op.Run(&buf)

		require.NoError(t, err)
		assert.Equal(t, "Deployment complete!\n", buf.String())
	})

	t.Run("Run writes default message when deployment_success_message is absent", func(t *testing.T) {
		dir := t.TempDir()
		composeFile := filepath.Join(dir, "compose.yaml")
		testutil.RequireWriteFile(t, composeFile, `
services:
  app:
    image: nginx
`)
		op := post_deploy.NewDeploySuccess(composeFile, command.LocalHost, "default message")
		var buf bytes.Buffer

		err := op.Run(&buf)

		require.NoError(t, err)
		assert.Equal(t, "default message\n", buf.String())
	})

	t.Run("Run returns error when compose file does not exist", func(t *testing.T) {
		op := post_deploy.NewDeploySuccess("nonexistent.yaml", command.LocalHost, "Run `topo ps` to see deployed containers")
		var buf bytes.Buffer

		err := op.Run(&buf)

		require.Error(t, err)
	})
}

func TestDefaultMessage(t *testing.T) {
	tests := []struct {
		name        string
		composeFile string
		want        string
	}{
		{
			name:        "compose yaml",
			composeFile: "compose.yaml",
			want:        "Run `topo ps` to see deployed containers",
		},
		{
			name:        "compose yml",
			composeFile: "compose.yml",
			want:        "Run `topo ps -f compose.yml` to see deployed containers",
		},
		{
			name:        "custom compose file",
			composeFile: "custom-compose.yaml",
			want:        "Run `topo ps -f custom-compose.yaml` to see deployed containers",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := post_deploy.DefaultMessage(tt.composeFile)

			assert.Equal(t, tt.want, got)
		})
	}
}
