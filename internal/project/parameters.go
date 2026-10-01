package project

import (
	"cmp"
	"fmt"
	"os"
	"slices"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"github.com/compose-spec/compose-go/v2/template"
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
	model, err := readUninterpolated(scope.ComposeFile)
	if err != nil {
		return nil, err
	}
	references := collectParameterReferences(model)
	for i := range parameters {
		param := &parameters[i]
		param.References = references[param.Name]
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
			Example:     param.Example,
		})
	}

	return parameters, nil
}

func collectParameterReferences(model map[string]any) map[string][]parameter.Reference {
	type node struct {
		path  string
		value any
	}
	references := map[string][]parameter.Reference{}
	pending := []node{{value: model}}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		switch value := current.value.(type) {
		case map[string]any:
			for key, child := range value {
				path := key
				if current.path != "" {
					path = current.path + "." + key
				}
				pending = append(pending, node{path: path, value: child})
			}
		case []any:
			for index, child := range value {
				path := fmt.Sprintf("%s[%d]", current.path, index)
				pending = append(pending, node{path: path, value: child})
			}
		case string:
			for name := range template.ExtractVariables(map[string]any{"reference": value}, nil) {
				references[name] = append(references[name], parameter.Reference{
					Path:       current.path,
					Expression: value,
				})
			}
		}
	}
	for _, usages := range references {
		slices.SortFunc(usages, func(a, b parameter.Reference) int {
			return cmp.Compare(a.Path, b.Path)
		})
	}
	return references
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
	Example     string `yaml:"example,omitempty"`
}
