package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/configs"
	"gopkg.in/yaml.v3"
)

type ServiceTemplateRepo struct {
	Id  string `json:"id"`
	Url string `json:"url"`
	Ref string `json:"ref,omitempty"`
}

type ServiceTemplateManifest struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Features    []string               `yaml:"features,omitempty"`
	Service     map[string]interface{} `yaml:"service"`
}

func loadTemplates() ([]ServiceTemplateRepo, error) {
	var templates []ServiceTemplateRepo
	dec := json.NewDecoder(bytes.NewReader(configs.ServiceTemplatesJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetTemplate(id string) (*ServiceTemplateRepo, error) {
	templates, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	for i := range templates {
		if templates[i].Id == id {
			return &templates[i], nil
		}
	}
	return nil, fmt.Errorf("Service Template with id %q not found", id)
}

func List() error {
	templates, err := loadTemplates()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

const TopoServiceFilename = "topo-service.yaml"

func ParseServiceDefinition(destDir string) (ServiceTemplateManifest, error) {
	topoServicePath := filepath.Join(destDir, TopoServiceFilename)
	topoServiceData, err := os.ReadFile(topoServicePath)
	if err != nil {
		return ServiceTemplateManifest{}, fmt.Errorf("failed to read %s from %s: %w", TopoServiceFilename, topoServicePath, err)
	}
	var topoService ServiceTemplateManifest
	if err := yaml.Unmarshal(topoServiceData, &topoService); err != nil {
		return ServiceTemplateManifest{}, fmt.Errorf("failed to parse %s: %w", TopoServiceFilename, err)
	}
	return topoService, nil
}
