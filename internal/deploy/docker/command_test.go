package docker_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/project"
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
	t.Run("passes compose files in the specified order", func(t *testing.T) {
		host := docker.NewHostFromDestination(ssh.NewDestination("localhost"))
		scope := project.Scope{ComposeFiles: []string{"base.yaml", "override.yaml"}}

		cmd := docker.ComposeCommand(t.Context(), host, scope, "up", "-d")

		want := []string{"docker", "compose", "-f", "base.yaml", "-f", "override.yaml", "up", "-d"}
		assert.Equal(t, want, cmd.Args)
	})

	t.Run("sets host flag for remote host", func(t *testing.T) {
		dest := ssh.NewDestination("ssh://user@remote")
		remoteHost := docker.NewHostFromDestination(dest)
		scope := project.Scope{ComposeFiles: []string{"/path/to/compose.yaml"}}

		cmd := docker.ComposeCommand(t.Context(), remoteHost, scope, "up", "-d")

		want := []string{"docker", "-H", "ssh://user@remote", "compose", "-f", "/path/to/compose.yaml", "up", "-d"}
		assert.Equal(t, want, cmd.Args)
	})

	t.Run("sets command env vars", func(t *testing.T) {
		dest := ssh.NewDestination("localhost")
		remoteHost := docker.NewHostFromDestination(dest)
		scope := project.Scope{ComposeFiles: []string{"compose.yaml"}, Env: []string{"FOO=NOTBAR"}}

		cmd := docker.ComposeCommand(t.Context(), remoteHost, scope, "up", "-d")

		assert.Contains(t, cmd.Env, "FOO=NOTBAR")
	})

	t.Run("sets env-file args", func(t *testing.T) {
		dest := ssh.NewDestination("localhost")
		remoteHost := docker.NewHostFromDestination(dest)
		scope := project.Scope{ComposeFiles: []string{"compose.yaml"}, EnvFiles: []string{".foobar.env", ".env"}}

		cmd := docker.ComposeCommand(t.Context(), remoteHost, scope, "up", "-d")

		want := []string{"docker", "compose", "-f", "compose.yaml", "--env-file", ".foobar.env", "--env-file", ".env", "up", "-d"}
		assert.Equal(t, want, cmd.Args)
	})
}
