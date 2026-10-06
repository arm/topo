package deploy_test

import (
	"context"
	"io"
	"testing"

	"github.com/arm/topo/internal/deploy"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullImages(t *testing.T) {
	t.Run("skips services that have a build context", func(t *testing.T) {
		var calledWith []string
		composeRunner := func(_ context.Context, _ io.Writer, _ project.Scope, args ...string) error {
			calledWith = args
			return nil
		}

		composeFilePath := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  locally-built:
    build:
      context: .
      dockerfile_inline: "FROM alpine:latest"
    image: this-image-does-not-exist-on-docker-hub
  to-pull:
    image: whatever
`)
		err := deploy.PullImages(
			t.Context(),
			term.NewProgress(io.Discard),
			project.Scope{ComposeFile: composeFilePath},
			composeRunner,
		)

		require.NoError(t, err)
		want := []string{"pull", "to-pull"}
		assert.Equal(t, want, calledWith)
	})
}
