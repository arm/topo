package template

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ComposeFilename = "compose.yaml"

type Template struct {
	Metadata    Metadata
	Service     map[string]any
	ServiceName string
}

type Metadata struct {
	Name        string
	Description string
	Features    []string
	Args        []Arg
}

type Arg struct {
	Name        string
	Description string
	Required    bool
	Example     string
}

type ComposeFile struct {
	Services map[string]any `yaml:"services"`
	XTopo    Metadata       `yaml:"x-topo"`
}

func ParseDefinition(destDir string) (ComposeFile, error) {
	composeServicePath := filepath.Join(destDir, ComposeFilename)
	composeServiceData, err := os.ReadFile(composeServicePath)
	if err != nil {
		return ComposeFile{}, fmt.Errorf("failed to read %s from %s: %w", ComposeFilename, composeServicePath, err)
	}

	var parsed ComposeFile
	if err := yaml.Unmarshal(composeServiceData, &parsed); err != nil {
		return ComposeFile{}, fmt.Errorf("failed to parse %s: %w", ComposeFilename, err)
	}

	return parsed, nil
}

func ParseComposeFileToTemplates(destDir string) ([]Template, error) {
	parsed, err := ParseDefinition(destDir)
	if err != nil {
		return nil, err
	}

	var templates []Template
	for name, svc := range parsed.Services {
		templates = append(templates, Template{
			Metadata:    parsed.XTopo,
			Service:     svc.(map[string]any),
			ServiceName: name,
		})
	}

	return templates, nil
}

type rawMetadata struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Features    []string          `yaml:"features,omitempty"`
	Args        map[string]rawArg `yaml:"args,omitempty"`
}

type rawArg struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Example     string `yaml:"example,omitempty"`
}

func (t *Metadata) UnmarshalYAML(node *yaml.Node) error {
	var raw rawMetadata
	if err := node.Decode(&raw); err != nil {
		return err
	}

	t.Name = raw.Name
	t.Description = raw.Description
	t.Features = raw.Features
	t.Args = parseArgsInOrder(node, raw.Args)

	return nil
}

func parseArgsInOrder(node *yaml.Node, argsMap map[string]rawArg) []Arg {
	var result []Arg

	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "args" {
			argsNode := node.Content[i+1]
			for j := 0; j < len(argsNode.Content); j += 2 {
				name := argsNode.Content[j].Value
				if metadata, ok := argsMap[name]; ok {
					result = append(result, Arg{
						Name:        name,
						Description: metadata.Description,
						Required:    metadata.Required,
						Example:     metadata.Example,
					})
				}
			}
			break
		}
	}

	return result
}
