package project

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"gopkg.in/yaml.v3"
)

func LoadParameters(scope Scope) ([]parameter.Parameter, error) {
	parameters, err := loadParameterMetadata(scope.ComposeFile)
	if err != nil {
		return nil, err
	}
	if len(parameters) == 0 {
		return parameters, nil
	}

	currentValues, err := env.CurrentValues(scope.EnvFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load current environment values: %w", err)
	}
	for i := range parameters {
		param := &parameters[i]
		if value, present := currentValues[param.Name]; present {
			param.ExistingValue = &value
		}
	}
	return parameters, nil
}

func loadParameterMetadata(composeFilePath string) ([]parameter.Parameter, error) {
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

	parameters := make([]parameter.Parameter, 0, len(raw.Metadata.Parameters))
	for name, param := range raw.Metadata.Parameters {
		parameters = append(parameters, parameter.Parameter{
			Name:        name,
			Description: param.Description,
			Required:    param.Required,
			Example:     param.Example,
		})
	}

	return parameters, nil
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
