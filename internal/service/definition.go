package service

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type TemplateManifest struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Features    []string       `yaml:"features,omitempty"`
	Service     map[string]any `yaml:"service"`
}

const TopoServiceFilename = "topo-service.yaml"

func ParseDefinition(destDir string) (TemplateManifest, error) {
	topoServicePath := filepath.Join(destDir, TopoServiceFilename)
	topoServiceData, err := os.ReadFile(topoServicePath)
	if err != nil {
		return TemplateManifest{}, fmt.Errorf("failed to read %s from %s: %w", TopoServiceFilename, topoServicePath, err)
	}
	var topoService TemplateManifest
	if err := yaml.Unmarshal(topoServiceData, &topoService); err != nil {
		return TemplateManifest{}, fmt.Errorf("failed to parse %s: %w", TopoServiceFilename, err)
	}
	return topoService, nil
}
