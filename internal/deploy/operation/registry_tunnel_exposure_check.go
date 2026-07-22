package operation

import (
	"context"
	"io"

	"github.com/arm/topo/internal/deploy/docker"
	"github.com/arm/topo/internal/ssh"
)

type RegistryTunnelExposureCheck struct {
	TargetDest ssh.Destination
	Port       string
}

// NewRegistryTunnelExposureCheck checks whether the registry tunnel port exposes
// the registry to the target's network, rather than being limited to target
// loopback. This can happen when the SSH server permits non-loopback remote
// forwards.
func NewRegistryTunnelExposureCheck(targetDest ssh.Destination, port string) *RegistryTunnelExposureCheck {
	return &RegistryTunnelExposureCheck{TargetDest: targetDest, Port: port}
}

func (c *RegistryTunnelExposureCheck) Description() string {
	return "Check registry tunnel is not exposed on remote network"
}

func (c *RegistryTunnelExposureCheck) Run(output io.Writer) error {
	return docker.CheckTunnelExposure(context.Background(), output, c.TargetDest, c.Port)
}
