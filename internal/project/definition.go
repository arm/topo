package project

import (
	"fmt"
	"io"

	"github.com/arm/topo/internal/output/logger"
	"gopkg.in/yaml.v3"
)

const ComposeFilename = "compose.yaml"

type Project struct {
	Metadata Metadata
}

type Metadata struct {
	Name        string
	Description string
	Features    []string
	Parameters  []Parameter
}

type Parameter struct {
	Name        string
	Description string
	Required    bool
	Example     string
	Default     string
}

func FromContent(reader io.Reader) (Project, error) {
	type composeFile struct {
		XTopo Metadata `yaml:"x-topo"`
	}

	var parsed composeFile
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&parsed); err != nil {
		return Project{}, fmt.Errorf("failed to decode project: %w", err)
	}

	return Project{
		Metadata: parsed.XTopo,
	}, nil
}

type rawMetadata struct {
	Name        string                  `yaml:"name"`
	Description string                  `yaml:"description"`
	Features    []string                `yaml:"features,omitempty"`
	Parameters  map[string]rawParameter `yaml:"parameters,omitempty"`
	Args        map[string]rawParameter `yaml:"args,omitempty"`
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
	t.Features = raw.Features
	parametersNode := findMetadataNode(node, "parameters")
	parameters := raw.Parameters
	if len(parameters) == 0 && len(raw.Args) > 0 {
		logger.Warn("x-topo.args is deprecated; use x-topo.parameters instead")
		parametersNode = findMetadataNode(node, "args")
		parameters = raw.Args
	}
	t.Parameters = parseParametersInOrder(parametersNode, parameters)

	return nil
}

func findMetadataNode(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func parseParametersInOrder(parametersNode *yaml.Node, parametersMap map[string]rawParameter) []Parameter {
	var result []Parameter
	parametersNode = resolveAlias(parametersNode)
	if parametersNode == nil {
		return result
	}

	for i := 0; i < len(parametersNode.Content); i += 2 {
		name := parametersNode.Content[i].Value
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

	return result
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.AliasNode {
		return node.Alias
	}

	return node
}
