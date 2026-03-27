package command_test

import (
	"testing"

	"github.com/arm/topo/internal/command"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	t.Run("converts docker command to string", func(t *testing.T) {
		h := ssh.NewDestination("user@remote")
		cmd := command.Docker(h, "save", "alpine:latest")

		got := command.String(cmd)

		want := "docker -H ssh://user@remote save alpine:latest"
		assert.Equal(t, want, got)
	})

	t.Run("converts docker compose command to string", func(t *testing.T) {
		h := ssh.NewDestination("user@remote")
		cmd := command.DockerCompose(h, "/path/to/compose.yaml", "up", "-d")

		got := command.String(cmd)

		want := "docker -H ssh://user@remote compose -f /path/to/compose.yaml up -d"
		assert.Equal(t, want, got)
	})
}

func TestWrapInLoginShell(t *testing.T) {
	t.Run("wraps command in login shell", func(t *testing.T) {
		got := command.WrapInLoginShell("echo $PATH")

		want := `/bin/sh -c "exec ${SHELL:-/bin/sh} -l -c \"echo \\\$PATH\""`
		assert.Equal(t, want, got)
	})
}
