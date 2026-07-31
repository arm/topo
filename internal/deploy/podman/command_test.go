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
		assert.Equal(t, "tcp://127.0.0.1:12345", environmentValue(command.Env, "CONTAINER_HOST"))
	})

	t.Run("does not configure a socket for localhost", func(t *testing.T) {
		command := podman.ComposeCommand(context.Background(), podman.LocalSocket, "info")

		assert.Empty(t, environmentValue(command.Env, "CONTAINER_HOST"))
	})
}

func TestComposeCommand(t *testing.T) {
	t.Run("sets args", func(t *testing.T) {
		command := podman.ComposeCommand(context.Background(), podman.LocalSocket, "compose.yaml", "up", "-d")

		assert.Equal(t, []string{"podman", "compose", "-f", "compose.yaml", "up", "-d"}, command.Args)
	})

	t.Run("configures compose provider", func(t *testing.T) {
		command := podman.ComposeCommand(context.Background(), podman.LocalSocket, "compose.yaml", "ps")

		assert.Equal(t, "docker-compose", environmentValue(command.Env, "PODMAN_COMPOSE_PROVIDER"))
	})

	t.Run("configures remote socket", func(t *testing.T) {
		t.Setenv("DOCKER_HOST", "unix:///stale-docker.sock")
		socket := podman.NewSocket("tcp://127.0.0.1:12345")

		command := podman.ComposeCommand(context.Background(), socket, "compose.yaml", "ps")

		assert.Equal(t, "tcp://127.0.0.1:12345", environmentValue(command.Env, "DOCKER_HOST"))
	})
}

func environmentValue(environment []string, key string) string {
	for index := len(environment) - 1; index >= 0; index-- {
		entry := environment[index]
		if len(entry) > len(key) && entry[:len(key)+1] == key+"=" {
			return entry[len(key)+1:]
		}
	}
	return ""
}
