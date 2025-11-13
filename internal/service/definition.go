package service

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type TemplateManifest struct {
	Metadata TopoMetadata
	Service  map[string]any
}

type TopoMetadata struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Features    []string               `yaml:"features,omitempty"`
	Args        map[string]ArgMetadata `yaml:"args,omitempty"`
}

type ArgMetadata struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Example     string `yaml:"example,omitempty"`
}

const ComposeServiceFilename = "compose.service.yaml"

type composeServiceFile struct {
	Services map[string]any `yaml:"services"`
	XTopo    TopoMetadata   `yaml:"x-topo"`
}

func ParseDefinition(destDir string) (TemplateManifest, error) {
	composeServicePath := filepath.Join(destDir, ComposeServiceFilename)
	composeServiceData, err := os.ReadFile(composeServicePath)
	if err != nil {
		return TemplateManifest{}, fmt.Errorf("failed to read %s from %s: %w", ComposeServiceFilename, composeServicePath, err)
	}

	var parsed composeServiceFile
	if err := yaml.Unmarshal(composeServiceData, &parsed); err != nil {
		return TemplateManifest{}, fmt.Errorf("failed to parse %s: %w", ComposeServiceFilename, err)
	}

	if len(parsed.Services) == 0 {
		return TemplateManifest{}, fmt.Errorf("no services defined in %s", ComposeServiceFilename)
	}

	if len(parsed.Services) > 1 {
		return TemplateManifest{}, fmt.Errorf("expected exactly one service in %s, found %d", ComposeServiceFilename, len(parsed.Services))
	}

	var serviceDef map[string]any
	for _, svc := range parsed.Services {
		serviceDef = svc.(map[string]any)
		break
	}

	return TemplateManifest{
		Metadata: parsed.XTopo,
		Service:  serviceDef,
	}, nil
}
