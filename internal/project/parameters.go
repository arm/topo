package project

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"gopkg.in/yaml.v3"
)

func LoadParameterDefinitions(composeFilePath string, currentValues map[string]string) ([]parameter.Definition, error) {
	reader, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open compose file: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	var raw parameterFile
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode project metadata: %w", err)
	}

	if len(raw.Metadata.Parameters) == 0 && len(raw.Metadata.Args) > 0 {
		logger.Warn("x-topo.args is deprecated; use x-topo.parameters instead")
		raw.Metadata.Parameters = raw.Metadata.Args
	}

	definitions := make([]parameter.Definition, 0, len(raw.Metadata.Parameters))
	for name, param := range raw.Metadata.Parameters {
		values := []string{}
		if val, ok := currentValues[name]; ok {
			values = append(values, val)
		}
		definitions = append(definitions, parameter.Definition{
			Name:          name,
			Description:   param.Description,
			Required:      param.Required,
			Example:       param.Example,
			CurrentValues: values,
		})
	}

	return definitions, nil
}

type parameterFile struct {
	Metadata parameterMetadata `yaml:"x-topo"`
}

type parameterMetadata struct {
	Parameters map[string]parameterFields `yaml:"parameters,omitempty"`
	Args       map[string]parameterFields `yaml:"args,omitempty"`
}

type parameterFields struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Example     string `yaml:"example,omitempty"`
}
