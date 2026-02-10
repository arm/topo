package describe

import (
	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

type HostCPU struct {
	Features []string
}

type RemoteprocCPU struct {
	Name string
}

type TargetHardwareReport struct {
	Host        HostCPU
	RemoteProcs []RemoteprocCPU
}

func Generate(sshTarget string) (TargetHardwareReport, error) {
	conn := health.NewConnection(sshTarget, ssh.ExecSSH)
	hwProfile, err := conn.ProbeHardware()
	if err != nil {
		return TargetHardwareReport{}, err
	}

	return generateReport(hwProfile), nil
}

func generateRemoteprocReport(remoteCPUs []string) []RemoteprocCPU {
	res := make([]RemoteprocCPU, len(remoteCPUs))
	for i, cpu := range remoteCPUs {
		res[i] = RemoteprocCPU{Name: cpu}
	}
	return res
}

func generateReport(hwProfile health.HardwareProfile) TargetHardwareReport {
	return TargetHardwareReport{
		Host: HostCPU{
			Features: hwProfile.Features,
		},
		RemoteProcs: generateRemoteprocReport(hwProfile.RemoteCPU),
	}
}
