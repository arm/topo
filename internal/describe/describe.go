package describe

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/target"
	"go.yaml.in/yaml/v4"
)

func ReadTargetDescriptionFromFile(filePath string) (*target.HardwareProfile, error) {
	description, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read target description file %q: %w", filePath, err)
	}

	var profile target.HardwareProfile
	if err := yaml.Unmarshal(description, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse target description file %q: %w", filePath, err)
	}
	return &profile, nil
}
