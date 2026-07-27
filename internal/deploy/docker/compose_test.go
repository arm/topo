package docker_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/stretchr/testify/require"
)

func TestPullImages(t *testing.T) {
	requireDocker(t)

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
		requireWriteFile(t, composeFilePath, composeFileContent)
		var output bytes.Buffer

		err := docker.PullImages(context.Background(), &output, composeFilePath, docker.LocalHost)

		require.NoError(t, err)
	})
}
