package operation_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/deploy/docker/operation"
	"github.com/arm/topo/internal/deploy/docker/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocker(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		testutil.RequireDocker(t)

		t.Run("executes docker command with args", func(t *testing.T) {
			var buf bytes.Buffer
			op := operation.NewDocker("Test docker version", command.LocalHost, []string{})

			err := op.Run(&buf)

			require.NoError(t, err)
			assert.Contains(t, buf.String(), "Docker version")
		})
	})

	t.Run("Description", func(t *testing.T) {
		t.Run("returns provided description", func(t *testing.T) {
			description := "Custom docker operation"
			op := operation.NewDocker(description, command.LocalHost, []string{})

			got := op.Description()

			assert.Equal(t, description, got)
		})
	})
}

func TestNewDockerPull(t *testing.T) {
	image := "nginx:latest"
	op := operation.NewDockerPull(command.LocalHost, image)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Pull image nginx:latest", got)
	})
}

func TestNewDockerStart(t *testing.T) {
	container := "my-container"
	op := operation.NewDockerStart(command.LocalHost, container)

	t.Run("Description", func(t *testing.T) {
		got := op.Description()

		assert.Equal(t, "Start container my-container", got)
	})
}

func TestNewDockerRun(t *testing.T) {
	image := "alpine:latest"
	container := "test-container"

	t.Run("Description", func(t *testing.T) {
		op := operation.NewDockerRun(command.LocalHost, image, container, []string{})

		got := op.Description()

		assert.Equal(t, "Run image alpine:latest as container test-container", got)
	})
}
