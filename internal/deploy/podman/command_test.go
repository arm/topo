package podman_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/deploy/podman"
	"github.com/stretchr/testify/assert"
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

func TestComposeCommand(t *testing.T) {
	t.Run("sets args", func(t *testing.T) {
		command := podman.ComposeCommand(context.Background(), podman.NewSocket("tcp://127.0.0.1:12345"), "compose.yaml", "up", "-d")
		assert.Equal(t, []string{"podman", "compose", "-f", "compose.yaml", "up", "-d"}, command.Args)
	})

	t.Run("configures compose provider", func(t *testing.T) {
		command := podman.ComposeCommand(context.Background(), podman.NewSocket("tcp://127.0.0.1:12345"), "compose.yaml", "ps")
		assert.Contains(t, command.Env, "PODMAN_COMPOSE_PROVIDER=podman-compose")
	})

	t.Run("configures remote socket", func(t *testing.T) {
		t.Setenv("CONTAINER_HOST", "unix:///stale-podman.sock")
		socket := podman.NewSocket("tcp://127.0.0.1:12345")

		command := podman.ComposeCommand(context.Background(), socket, "compose.yaml", "ps")
		assert.Contains(t, command.Env, "CONTAINER_HOST=tcp://127.0.0.1:12345")
	})
}
