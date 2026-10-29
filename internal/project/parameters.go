package project

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"github.com/compose-spec/compose-go/v2/template"
	"gopkg.in/yaml.v3"
)

func LoadParameters(scope Scope) ([]parameter.Parameter, error) {
	model, err := readUninterpolated(scope.ComposeFiles)
	if err != nil {
		return nil, err
	}
	parameters, err := decodeParameterMetadata(model)
	if err != nil {
		return nil, err
	}
	if len(parameters) == 0 {
		return parameters, nil
	}
	if usesLiteralBuildArgConfiguration(model, parameters) {
		return nil, ErrParameterMigrationRequired
	}

	currentValues, err := env.CurrentValues(scope.EnvFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load current environment values: %w", err)
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

func usesLiteralBuildArgConfiguration(model map[string]any, parameters []parameter.Parameter) bool {
	services, _ := model["services"].(map[string]any)
	var buildArguments []any
	argumentNames := map[string]bool{}
	for _, value := range services {
		service := value.(map[string]any)
		build, _ := service["build"].(map[string]any)
		arguments, _ := build["args"].(map[string]any)
		buildArguments = append(buildArguments, arguments)
		for name := range arguments {
			argumentNames[name] = true
		}
	}
	references := template.ExtractVariables(map[string]any{"args": buildArguments}, nil)
	hasParameterValues := false
	for _, param := range parameters {
		if _, referenced := references[param.Name]; referenced {
			return false
		}
		if argumentNames[param.Name] {
			hasParameterValues = true
		}
	}
	return hasParameterValues
}

func decodeParameterMetadata(model map[string]any) ([]parameter.Parameter, error) {
	var document yaml.Node
	if err := document.Encode(model); err != nil {
		return nil, fmt.Errorf("failed to encode project metadata: %w", err)
	}

	var raw parameterFile
	if err := document.Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode project metadata: %w", err)
	}

	if len(raw.Metadata.Parameters) == 0 && len(raw.Metadata.Args) > 0 {
		logger.Warn("x-topo.args is deprecated; use x-topo.parameters instead")
		raw.Metadata.Parameters = raw.Metadata.Args
	}

	parameters := make([]parameter.Parameter, 0, len(raw.Metadata.Parameters))
	for name, param := range raw.Metadata.Parameters {
		var source *parameter.Source
		if param.Source != nil {
			source = &parameter.Source{
				Provider: param.Source.Provider,
				Group:    param.Source.Group,
				Fact:     param.Source.Fact,
			}
		}
		parameters = append(parameters, parameter.Parameter{
			Name:        name,
			Description: param.Description,
			Example:     param.Example,
			Source:      source,
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
	Source      *sourceFields `yaml:"source,omitempty"`
	Description string        `yaml:"description"`
	Example     string        `yaml:"example,omitempty"`
}

type sourceFields struct {
	Provider string `yaml:"provider"`
	Group    string `yaml:"group"`
	Fact     string `yaml:"fact"`
}
