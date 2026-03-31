package command_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker/command"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestNewHostFromDestination(t *testing.T) {
	t.Run("gets new host from destination", func(t *testing.T) {
		dest := ssh.NewDestination("ssh://user@remote")

		got := command.NewHostFromDestination(dest)

		dontwant := command.LocalHost
		assert.NotEqual(t, dontwant, got)
	})

	t.Run("gets localhost when given localhost Destination", func(t *testing.T) {
		got := command.NewHostFromDestination(ssh.PlainLocalhost)

		want := command.LocalHost
		assert.Equal(t, want, got)
	})
}
