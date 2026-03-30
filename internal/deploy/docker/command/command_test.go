package command_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	t.Run("converts docker command to string", func(t *testing.T) {
		verifiedDest := ssh.NewDestination("ssh://user@remote")
		h := command.NewHostFromDestination(verifiedDest)
		cmd := command.Docker(h, "save", "alpine:latest")

		got := command.String(cmd)

		want := "docker -H ssh://user@remote save alpine:latest"
		assert.Equal(t, want, got)
	})

	t.Run("converts docker compose command to string", func(t *testing.T) {
		verifiedDest := ssh.NewDestination("ssh://user@remote")
		h := command.NewHostFromDestination(verifiedDest)
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

func TestBinaryLookupCommand(t *testing.T) {
	t.Run("returns wrapped command for valid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("docker")

		require.NoError(t, err)
		assert.Equal(t, command.UnsafeBinaryLookupCommand("docker"), got)
	})

	t.Run("returns error for invalid binary", func(t *testing.T) {
		got, err := command.BinaryLookupCommand("bad name")

		assert.Error(t, err)
		assert.Empty(t, got)
	})
}

func TestUnsafeBinaryLookupCommand(t *testing.T) {
	t.Run("returns wrapped command without validation", func(t *testing.T) {
		got := command.UnsafeBinaryLookupCommand("docker")

		assert.Equal(t, command.WrapInLoginShell("command -v docker"), got)
	})
}
