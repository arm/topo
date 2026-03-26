package docker_test

import (
	"testing"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/deploy/docker/operation"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/assert"
)

func TestNewDeploymentPs(t *testing.T) {
	composeFile := "compose.yaml"

	t.Run("runs ps operation for remote host", func(t *testing.T) {
		remoteHost := ssh.NewDestination("user@remote")

		got := docker.NewDeploymentPs(composeFile, remoteHost)

		want := goperation.Sequence{
			operation.NewDockerComposePs(composeFile, remoteHost),
		}
		assert.Equal(t, want, got)
	})

	t.Run("runs ps operation for local host", func(t *testing.T) {
		got := docker.NewDeploymentPs(composeFile, ssh.PlainLocalhost)

		want := goperation.Sequence{
			operation.NewDockerComposePs(composeFile, ssh.PlainLocalhost),
		}
		assert.Equal(t, want, got)
	})
}
