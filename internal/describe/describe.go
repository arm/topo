package describe

import (
	"github.com/arm-debug/topo-cli/internal/health"
	"github.com/arm-debug/topo-cli/internal/ssh"
)

type HostCPU struct {
	Features []string
	// TODO enrich with more details like CPU model and accessible memory
}

type RemoteProcCPU struct {
	Name string
}

type TargetHardwareReport struct {
	Host        HostCPU
	RemoteProcs []RemoteProcCPU
}

func Generate(sshTarget string) (TargetHardwareReport, error) {
	conn := health.NewConnection(sshTarget, ssh.ExecSSH)
	hwProfile, err := conn.ProbeHardware()
	if err != nil {
		return TargetHardwareReport{}, err
	}

	return generateReport(hwProfile), nil
}

func generateRemoteProcReport(remoteCPUs []string) []RemoteProcCPU {
	res := make([]RemoteProcCPU, len(remoteCPUs))
	for i, cpu := range remoteCPUs {
		res[i] = RemoteProcCPU{Name: cpu}
	}
	return res
}

func generateReport(hwProfile health.HardwareProfile) TargetHardwareReport {
	return TargetHardwareReport{
		Host: HostCPU{
			Features: hwProfile.Features,
		},
		RemoteProcs: generateRemoteProcReport(hwProfile.RemoteCPU),
	}
}
