package podman_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/arm/topo/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	t.Run("sets args", func(t *testing.T) {
		command := podman.Command(context.Background(), podman.LocalSocket, "info")

		assert.Equal(t, []string{"podman", "info"}, command.Args)
	})

	t.Run("configures remote socket", func(t *testing.T) {
		t.Setenv("CONTAINER_HOST", "unix:///stale-podman.sock")
		socket := podman.NewSocket("tcp://127.0.0.1:12345")

		command := podman.Command(context.Background(), socket, "info")

		assert.Equal(t, []string{"podman", "info"}, command.Args)
		assert.Contains(t, command.Env, "CONTAINER_HOST=tcp://127.0.0.1:12345")
	})
}

func TestComposeRawCommand(t *testing.T) {
	t.Run("uses the deployment Compose provider without overriding the socket", func(t *testing.T) {
		t.Setenv("DOCKER_HOST", "unix:///stale-docker.sock")
		t.Setenv("PODMAN_COMPOSE_PROVIDER", "yolo-swag")

		command := podman.ComposeRawCommand(t.Context(), "version")

		assert.Equal(t, []string{"podman", "compose", "version"}, command.Args)
		assert.Contains(t, command.Env, "PODMAN_COMPOSE_PROVIDER=docker-compose")
		assert.Contains(t, command.Env, "DOCKER_HOST=unix:///stale-docker.sock")
	})
}

func TestComposeSocketRawCommand(t *testing.T) {
	t.Run("uses the deployment Compose provider and socket configuration", func(t *testing.T) {
		t.Setenv("DOCKER_HOST", "unix:///stale-docker.sock")
		t.Setenv("PODMAN_COMPOSE_PROVIDER", "yolo-swag")
		socket := podman.NewSocket("tcp://127.0.0.1:12345")

		command, err := podman.ComposeSocketRawCommand(t.Context(), socket, "version")

		require.NoError(t, err)
		assert.Equal(t, []string{"podman", "compose", "version"}, command.Args)
		assert.Contains(t, command.Env, "PODMAN_COMPOSE_PROVIDER=docker-compose")
		assert.Contains(t, command.Env, "DOCKER_HOST=tcp://127.0.0.1:12345")
	})
}

func TestComposeCommand(t *testing.T) {
	t.Run("sets args", func(t *testing.T) {
		scope := project.Scope{ComposeFile: "compose.yaml"}

		command, err := podman.ComposeCommand(t.Context(), podman.NewSocket("tcp://127.0.0.1:12345"), scope, "up", "-d")

		require.NoError(t, err)
		assert.Equal(t, []string{"podman", "compose", "-f", "compose.yaml", "up", "-d"}, command.Args)
	})

	t.Run("sets command env vars", func(t *testing.T) {
		scope := project.Scope{ComposeFile: "compose.yaml", Env: []string{"FOO=BAR"}}

		command, err := podman.ComposeCommand(t.Context(), podman.NewSocket("tcp://127.0.0.1:12345"), scope, "up", "-d")

		require.NoError(t, err)
		assert.Contains(t, command.Env, "FOO=BAR")
	})

	t.Run("sets env-file args", func(t *testing.T) {
		scope := project.Scope{ComposeFile: "compose.yaml", EnvFiles: []string{".foobar.env", ".env"}}

		command, err := podman.ComposeCommand(t.Context(), podman.NewSocket("tcp://127.0.0.1:12345"), scope, "up", "-d")

		require.NoError(t, err)
		assert.Equal(t, []string{"podman", "compose", "-f", "compose.yaml", "--env-file", ".foobar.env", "--env-file", ".env", "up", "-d"}, command.Args)
	})

	t.Run("configures compose provider", func(t *testing.T) {
		command, err := podman.ComposeCommand(context.Background(), podman.NewSocket("tcp://127.0.0.1:12345"), project.Scope{ComposeFile: "compose.yaml"}, "ps")

		require.NoError(t, err)
		assert.Contains(t, command.Env, "PODMAN_COMPOSE_PROVIDER=docker-compose")
	})

	t.Run("configures remote socket", func(t *testing.T) {
		t.Setenv("DOCKER_HOST", "unix:///stale-docker.sock")
		socket := podman.NewSocket("tcp://127.0.0.1:12345")

		command, err := podman.ComposeCommand(context.Background(), socket, project.Scope{ComposeFile: "compose.yaml"}, "ps")

		require.NoError(t, err)
		assert.Contains(t, command.Env, "DOCKER_HOST=tcp://127.0.0.1:12345")
	})
}
