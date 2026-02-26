package health

import (
	"github.com/arm/topo/internal/ssh"
	"github.com/arm/topo/internal/target"
)

type HardwareProfile struct {
	RemoteCPU []target.RemoteprocCPU
}

func (h HardwareProfile) Capabilities() map[HardwareCapability]struct{} {
	capabilities := make(map[HardwareCapability]struct{})
	if len(h.RemoteCPU) > 0 {
		capabilities[Remoteproc] = struct{}{}
	}
	return capabilities
}

type Status struct {
	SSHTarget       ssh.Host
	ConnectionError error
	Dependencies    []DependencyStatus
	Hardware        HardwareProfile
}

func ProbeHealthStatus(c target.Connection) Status {
	var status Status
	status.SSHTarget = c.SSHTarget

	if err := c.ProbeAuthentication(); err != nil {
		status.ConnectionError = err
		return status
	}

	remoteprocs, _ := c.ProbeRemoteproc()
	status.Hardware.RemoteCPU = remoteprocs
	status.Dependencies = CheckDependencies(c.BinaryExists, status.Hardware.Capabilities())

	return status
}
