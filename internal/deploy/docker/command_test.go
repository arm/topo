package docker_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	t.Run("builds docker command for remote host", func(t *testing.T) {
		dest := ssh.NewDestination("ssh://user@remote")
		remoteHost := docker.NewHostFromDestination(dest)

		cmd := docker.Command(t.Context(), remoteHost, "save", "alpine:latest")

		want := []string{"docker", "-H", "ssh://user@remote", "save", "alpine:latest"}
		assert.Equal(t, want, cmd.Args)
	})
}

func TestComposeCommand(t *testing.T) {
	t.Run("builds docker compose command for remote host", func(t *testing.T) {
		dest := ssh.NewDestination("ssh://user@remote")
		remoteHost := docker.NewHostFromDestination(dest)

		cmd := docker.ComposeCommand(t.Context(), remoteHost, "/path/to/compose.yaml", "up", "-d")

		want := []string{"docker", "-H", "ssh://user@remote", "compose", "-f", "/path/to/compose.yaml", "up", "-d"}
		assert.Equal(t, want, cmd.Args)
	})
}
