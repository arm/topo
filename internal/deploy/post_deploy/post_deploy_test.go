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
		op := post_deploy.NewDeploySuccess(composeFile, command.LocalHost, "Run `topo ps` to see deployed containers")
		var buf bytes.Buffer

		err := op.Run(&buf)

		require.NoError(t, err)
		assert.Equal(t, "Run `topo ps` to see deployed containers\n", buf.String())
	})

	t.Run("Run returns error when compose file does not exist", func(t *testing.T) {
		op := post_deploy.NewDeploySuccess("nonexistent.yaml", command.LocalHost, "Run `topo ps` to see deployed containers")
		var buf bytes.Buffer

		err := op.Run(&buf)

		require.Error(t, err)
	})
}
