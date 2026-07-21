package docker_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/command"
	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/stretchr/testify/require"
)

func TestPullImages(t *testing.T) {
	testutil.RequireDocker(t)

	t.Run("skips services that have a build context", func(t *testing.T) {
		composeFilePath := filepath.Join(t.TempDir(), "compose.yaml")
		composeFileContent := `
services:
  locally-built:
    build:
      context: .
      dockerfile_inline: "FROM alpine:latest"
    image: this-image-does-not-exist-on-docker-hub
`
		testutil.RequireWriteFile(t, composeFilePath, composeFileContent)
		var output bytes.Buffer

		err := docker.PullImages(context.Background(), &output, composeFilePath, command.LocalHost)

		require.NoError(t, err)
	})
}
