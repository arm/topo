package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ComposeFilename = "compose.yaml"

type Template struct {
	Metadata Metadata
	Services []Service
}

type Service struct {
	Name string
	Data map[string]any
}

type Metadata struct {
	Name                     string
	Description              string
	DeploymentSuccessMessage string
	Features                 []string
	Parameters               []Parameter
}

type Parameter struct {
	Name        string
	Description string
	Required    bool
	Example     string
	Default     string
}

func FromContent(reader io.Reader) (Template, error) {
	type composeFile struct {
		Services map[string]any `yaml:"services"`
		XTopo    Metadata       `yaml:"x-topo"`
	}

	var parsed composeFile
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&parsed); err != nil {
		return Template{}, fmt.Errorf("failed to decode template: %w", err)
	}

	var services []Service
	for name, svc := range parsed.Services {
		services = append(services, Service{
			Data: svc.(map[string]any),
			Name: name,
		})
	}

	return Template{
		Services: services,
		Metadata: parsed.XTopo,
	}, nil
}

func FromDir(destDir string) (Template, error) {
	composeServicePath := filepath.Join(destDir, ComposeFilename)

	f, err := os.Open(composeServicePath)
	if err != nil {
		return Template{}, err
	}
	defer func() { _ = f.Close() }()

	return FromContent(f)
}

type rawMetadata struct {
	Name                     string                  `yaml:"name"`
	Description              string                  `yaml:"description"`
	DeploymentSuccessMessage string                  `yaml:"deployment_success_message"`
	Features                 []string                `yaml:"features,omitempty"`
	Parameters               map[string]rawParameter `yaml:"parameters,omitempty"`
}

type rawParameter struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Example     string `yaml:"example,omitempty"`
	Default     string `yaml:"default,omitempty"`
}

func (t *Metadata) UnmarshalYAML(node *yaml.Node) error {
	var raw rawMetadata
	if err := node.Decode(&raw); err != nil {
		return err
	}

	t.Name = raw.Name
	t.Description = raw.Description
	t.DeploymentSuccessMessage = raw.DeploymentSuccessMessage
	t.Features = raw.Features
	t.Parameters = parseParametersInOrder(node, raw.Parameters)

	return nil
}

func parseParametersInOrder(node *yaml.Node, parametersMap map[string]rawParameter) []Parameter {
	var result []Parameter

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "parameters" {
			parametersNode := node.Content[i+1]
			for j := 0; j < len(parametersNode.Content); j += 2 {
				name := parametersNode.Content[j].Value
				if metadata, ok := parametersMap[name]; ok {
					result = append(result, Parameter{
						Name:        name,
						Description: metadata.Description,
						Required:    metadata.Required,
						Example:     metadata.Example,
						Default:     metadata.Default,
					})
				}
			}
			break
		}
	}

	return result
}
