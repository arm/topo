package views

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/arm/topo/cli/internal/probe"
	"go.yaml.in/yaml/v4"
)

type TargetDescription struct {
	probe.HardwareProfile
}

func (d TargetDescription) AsPlain(_ bool) (string, error) {
	var buf bytes.Buffer
	if err := yaml.NewEncoder(&buf).Encode(d.HardwareProfile); err != nil {
		return "", fmt.Errorf("encode target description as yaml: %w", err)
	}
	return buf.String(), nil
}

func (d TargetDescription) AsJSON() (string, error) {
	b, err := json.MarshalIndent(d.HardwareProfile, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode target description as json: %w", err)
	}
	return string(b), nil
}
