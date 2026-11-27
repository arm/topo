package post_deploy_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/arm/topo/internal/deploy/post_deploy"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintDeploySuccess(t *testing.T) {
	t.Run("writes deployment_success_message from compose file", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
x-topo:
  deployment_success_message: "Deployment complete!"
services:
  app:
    image: nginx
`)
		var buf bytes.Buffer

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := post_deploy.PrintDeploySuccess(&buf, scope, "Run `topo ps` to see deployed containers")

		require.NoError(t, err)
		assert.Equal(t, "Deployment complete!\n", buf.String())
	})

	t.Run("writes default message when deployment_success_message is absent", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  app:
    image: nginx
`)
		var buf bytes.Buffer

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := post_deploy.PrintDeploySuccess(&buf, scope, "default message")

		require.NoError(t, err)
		assert.Equal(t, "default message\n", buf.String())
	})

	t.Run("interpolates env vars in deployment_success_message", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), `
name: test-project
x-topo:
  deployment_success_message: "${COMPOSE_PROJECT_NAME} deployed - ${EXTRA_MESSAGE}"
services:
  app:
    image: nginx
`)
		var buf bytes.Buffer
		t.Setenv("EXTRA_MESSAGE", "cool!")

		scope, prepareErr := project.BuildScope(composeFile, "", nil)
		require.NoError(t, prepareErr)

		err := post_deploy.PrintDeploySuccess(&buf, scope, "default message")

		require.NoError(t, err)
		assert.Equal(t, "test-project deployed - cool!\n", buf.String())
	})

	t.Run("returns error when compose file is removed after preparation", func(t *testing.T) {
		composeFile := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")
		scope, err := project.BuildScope(composeFile, "", nil)
		require.NoError(t, err)
		require.NoError(t, os.Remove(composeFile))
		var buf bytes.Buffer

		err = post_deploy.PrintDeploySuccess(&buf, scope, "Run `topo ps` to see deployed containers")

		require.Error(t, err)
	})
}
