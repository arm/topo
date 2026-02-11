package describe

import (
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/health"
	"go.yaml.in/yaml/v4"
)

const TargetDescriptionFilename = "target-description.yaml"

func GenerateTargetDescription(conn health.Connection) (health.HardwareProfile, error) {
	if err := conn.ProbeConnection(); err != nil {
		return health.HardwareProfile{}, err
	}

	hwProfile, err := conn.ProbeHardware()
	if err != nil {
		return health.HardwareProfile{}, err
	}

	return hwProfile, nil
}

func WriteTargetDescriptionToFile(dir string, report health.HardwareProfile) (string, error) {
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
