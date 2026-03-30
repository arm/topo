package dockercommand_test

import (
	"testing"

	dockercommand "github.com/arm/topo/internal/deploy/docker/docker_command"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	t.Run("converts docker command to string", func(t *testing.T) {
		verifiedDest := ssh.NewDestination("ssh://user@remote")
		h := dockercommand.NewHostFromDestination(verifiedDest)
		cmd := dockercommand.Docker(h, "save", "alpine:latest")

		got := dockercommand.String(cmd)

		want := "docker -H ssh://user@remote save alpine:latest"
		assert.Equal(t, want, got)
	})

	t.Run("converts docker compose command to string", func(t *testing.T) {
		verifiedDest := ssh.NewDestination("ssh://user@remote")
		h := dockercommand.NewHostFromDestination(verifiedDest)
		cmd := dockercommand.DockerCompose(h, "/path/to/compose.yaml", "up", "-d")

		got := dockercommand.String(cmd)

		want := "docker -H ssh://user@remote compose -f /path/to/compose.yaml up -d"
		assert.Equal(t, want, got)
	})
}
