package docker_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestNewHostFromDestination(t *testing.T) {
	t.Run("gets new host from destination", func(t *testing.T) {
		dest := ssh.NewDestination("ssh://user@remote")

		got := docker.NewHostFromDestination(dest)

		dontwant := docker.LocalHost
		assert.NotEqual(t, dontwant, got)
	})

	t.Run("gets localhost when given localhost Destination", func(t *testing.T) {
		got := docker.NewHostFromDestination(ssh.PlainLocalhost)

		want := docker.LocalHost
		assert.Equal(t, want, got)
	})
}
