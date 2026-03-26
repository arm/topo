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
	SSHTarget       ssh.Destination
	ConnectionError error
	Dependencies    []DependencyStatus
	Hardware        HardwareProfile
}

func ProbeHealthStatus(c target.Connection, probeOpts target.SSHAuthenticationProbeOptions) Status {
	var status Status
	status.SSHTarget = c.SSHTarget

	authProbe := target.NewSSHAuthenticationProbe(&c, probeOpts)
	if err := authProbe.Probe(); err != nil {
		status.ConnectionError = err
		return status
	}

	probe := target.NewHardwareProbe(&c)
	remoteprocs, _ := probe.ProbeRemoteproc()
	status.Hardware.RemoteCPU = remoteprocs
	dependenciesToCheck := FilterByHardware(TargetRequiredDependencies, status.Hardware.Capabilities())
	status.Dependencies = PerformChecks(dependenciesToCheck, c.BinaryExists, func(fullCmd string) error {
		_, err := c.Run(fullCmd)
		return err
	})

	return status
}
