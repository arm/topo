package describe

import (
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/health"
	"go.yaml.in/yaml/v4"
)

const TargetDescriptionFilename = "target-description.yaml"

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

func Generate(conn health.Connection) (TargetHardwareReport, error) {
	if err := conn.ProbeConnection(); err != nil {
		return TargetHardwareReport{}, err
	}

	hwProfile, err := conn.ProbeHardware()
	if err != nil {
		return TargetHardwareReport{}, err
	}

	return generateReport(hwProfile), nil
}

func WriteTargetDescriptionFile(dir string, report TargetHardwareReport) (string, error) {
	outputFile := filepath.Join(dir, TargetDescriptionFilename)
	f, err := os.Create(outputFile)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()

	encoder := yaml.NewEncoder(f)
	return outputFile, encoder.Encode(report)
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
