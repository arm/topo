package describe

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/health"
	"go.yaml.in/yaml/v4"
)

const TargetDescriptionFilename = "target-description.yaml"

type TargetHostCPU struct {
	Features []string
}

type RemoteprocCPU struct {
	Name string
}

type TargetHardwareReport struct {
	Host        TargetHostCPU
	RemoteProcs []RemoteprocCPU
}

func GenerateTargetDescription(conn health.Connection) (TargetHardwareReport, error) {
	if err := conn.ProbeConnection(); err != nil {
		return TargetHardwareReport{}, err
	}

	hwProfile, err := conn.ProbeHardware()
	if err != nil {
		return TargetHardwareReport{}, err
	}

	return generateReport(hwProfile), nil
}

func WriteTargetDescriptionToFile(dir string, report TargetHardwareReport) (string, error) {
	outputFile := filepath.Join(dir, TargetDescriptionFilename)
	f, err := os.Create(outputFile)
	if err != nil {
		return "", err
	}
	encoder := yaml.NewEncoder(f)
	if err := encoder.Encode(report); err != nil {
		closeErr := f.Close()
		return "", errors.Join(err, closeErr)
	}
	return outputFile, f.Close()
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
		Host: TargetHostCPU{
			Features: hwProfile.Features,
		},
		RemoteProcs: generateRemoteprocReport(hwProfile.RemoteCPU),
	}
}
