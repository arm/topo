package service

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ComposeServiceFilename = "compose.service.yaml"

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

func ParseDefinition(destDir string) (Template, error) {
	type composeServiceFile struct {
		Services map[string]any `yaml:"services"`
		XTopo    Metadata       `yaml:"x-topo"`
	}

	composeServicePath := filepath.Join(destDir, ComposeServiceFilename)
	composeServiceData, err := os.ReadFile(composeServicePath)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read %s from %s: %w", ComposeServiceFilename, composeServicePath, err)
	}

	var parsed composeServiceFile
	if err := yaml.Unmarshal(composeServiceData, &parsed); err != nil {
		return Template{}, fmt.Errorf("failed to parse %s: %w", ComposeServiceFilename, err)
	}

	if len(parsed.Services) == 0 {
		return Template{}, fmt.Errorf("no services defined in %s", ComposeServiceFilename)
	}

	if len(parsed.Services) > 1 {
		return Template{}, fmt.Errorf("expected exactly one service in %s, found %d", ComposeServiceFilename, len(parsed.Services))
	}

	var serviceDef map[string]any
	var serviceName string
	for svcName, svc := range parsed.Services {
		serviceDef = svc.(map[string]any)
		serviceName = svcName
		break
	}

	return Template{
		Metadata:    parsed.XTopo,
		Service:     serviceDef,
		ServiceName: serviceName,
	}, nil
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
