package project

import (
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"gopkg.in/yaml.v3"
)

const ComposeFilename = "compose.yaml"

type Project struct {
	Metadata               Metadata
	currentParameterValues map[string][]string
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
}

func FromContent(reader io.Reader) (Project, error) {
	type composeFile struct {
		XTopo Metadata `yaml:"x-topo"`
	}

	var document yaml.Node
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&document); err != nil {
		return Project{}, fmt.Errorf("failed to decode project: %w", err)
	}
	if len(document.Content) == 0 {
		return Project{}, fmt.Errorf("failed to decode project: compose file is empty")
	}

	var parsed composeFile
	if err := document.Decode(&parsed); err != nil {
		return Project{}, fmt.Errorf("failed to decode project: %w", err)
	}

	return Project{
		Metadata:               parsed.XTopo,
		currentParameterValues: parseCurrentParameterValues(document.Content[0]),
	}, nil
}

type rawMetadata struct {
	Name                     string                  `yaml:"name"`
	Description              string                  `yaml:"description"`
	DeploymentSuccessMessage string                  `yaml:"deployment_success_message"`
	Features                 []string                `yaml:"features,omitempty"`
	Parameters               map[string]rawParameter `yaml:"parameters,omitempty"`
	Args                     map[string]rawParameter `yaml:"args,omitempty"`
}

type rawParameter struct {
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
	t.DeploymentSuccessMessage = raw.DeploymentSuccessMessage
	t.Features = raw.Features
	parametersNode := findMappingValue(node, "parameters")
	parameters := raw.Parameters
	if len(parameters) == 0 && len(raw.Args) > 0 {
		logger.Warn("x-topo.args is deprecated; use x-topo.parameters instead")
		parametersNode = findMappingValue(node, "args")
		parameters = raw.Args
	}
	t.Parameters = parseParametersInOrder(parametersNode, parameters)

	return nil
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return resolveAlias(node.Content[i+1])
		}
	}

	return nil
}

func parseCurrentParameterValues(root *yaml.Node) map[string][]string {
	values := make(map[string][]string)
	services := findMappingValue(root, "services")
	if services == nil {
		return values
	}

	for i := 0; i < len(services.Content); i += 2 {
		build := findMappingValue(services.Content[i+1], "build")
		args := findMappingValue(build, "args")
		if args == nil {
			continue
		}

		switch args.Kind {
		case yaml.MappingNode:
			for j := 0; j < len(args.Content); j += 2 {
				name := args.Content[j].Value
				value := resolveAlias(args.Content[j+1]).Value
				values[name] = append(values[name], value)
			}
		case yaml.SequenceNode:
			for _, node := range args.Content {
				name, value, _ := strings.Cut(resolveAlias(node).Value, "=")
				values[name] = append(values[name], value)
			}
		}
	}

	return values
}

func parseParametersInOrder(parametersNode *yaml.Node, parametersMap map[string]rawParameter) []Parameter {
	var result []Parameter
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
