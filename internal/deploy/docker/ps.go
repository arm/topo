package docker

import (
	"github.com/arm/topo/internal/deploy/docker/operation"
	goperation "github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/ssh"
)

func NewDeploymentPs(composeFile string, dest ssh.Destination) goperation.Sequence {
	return goperation.Sequence{operation.NewDockerComposePs(composeFile, dest)}
}
