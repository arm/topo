package env_test

import (
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestComposeEnv(t *testing.T) {
	target := ssh.NewDestination("user@localhost")

	got := env.ComposeEnv(target)

	want := []string{
		"TOPO_TARGET=ssh://user@localhost",
		"TOPO_TARGET_HOSTNAME=localhost",
	}
	assert.Equal(t, want, got)
}
