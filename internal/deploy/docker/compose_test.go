package docker_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestPullImages(t *testing.T) {
	requireDocker(t)

	t.Run("skips services that have a build context", func(t *testing.T) {
		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  locally-built:
    build:
      context: .
      dockerfile_inline: "FROM alpine:latest"
    image: this-image-does-not-exist-on-docker-hub
`)
		var output bytes.Buffer

		err := docker.PullImages(context.Background(), &output, docker.LocalHost, composeFilePath)

		require.NoError(t, err)
	})
}
