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
	root, err := readRootSource(scope)
	if err != nil {
		return nil, err
	}
	sources, err := discoverIncludeSources([]composeSource{root})
	if err != nil {
		return nil, err
	}
	parameters, err := loadParameterDeclarations(sources)
	if err != nil {
		return nil, err
	}
	if len(parameters) == 0 {
		return parameters, nil
	}
	for index := len(sources) - 1; index >= 0; index-- {
		if err := resolveSourceInheritance(&sources[index], sources); err != nil {
			return nil, err
		}
	}
	currentValues, err := env.CurrentValues(scope.EnvFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to load current environment values: %w", err)
	}
	references := collectSourceParameterReferences(sources)
	for i := range parameters {
		param := &parameters[i]
		param.References = references[param.Name]
		if value, present := currentValues[param.Name]; present {
			param.ExistingValue = &value
		}
	}
	return parameters, nil
}

func collectSourceParameterReferences(sources []composeSource) map[string][]parameter.Reference {
	references := map[string][]parameter.Reference{}
	for _, source := range sources {
		for name, usages := range collectParameterReferences(source.model) {
			for index := range usages {
				usages[index].Environment = source.environment
				if environment, present := source.referenceEnvironments[usages[index].Path]; present {
					usages[index].Environment = environment
				}
			}
			references[name] = append(references[name], usages...)
		}
	}
	return references
}

func loadParameterDeclarations(sources []composeSource) ([]parameter.Parameter, error) {
	var parameters []parameter.Parameter
	seen := map[string]bool{}
	for _, source := range sources {
		declared, err := decodeParameterMetadata(source.model)
		if err != nil {
			return nil, err
		}
		for _, param := range declared {
			if seen[param.Name] {
				return nil, fmt.Errorf("duplicate Topo parameter %q in %q", param.Name, source.path)
			}
			seen[param.Name] = true
			parameters = append(parameters, param)
		}
	}
	return parameters, nil
}

func selectFilePathParameters(parameters []parameter.Parameter, sources []composeSource, currentValues map[string]string) []parameter.Parameter {
	references := map[string][]parameter.Reference{}
	for _, source := range sources {
		for name, usages := range collectPathParameterReferences(source) {
			references[name] = append(references[name], usages...)
		}
	}
	var selected []parameter.Parameter
	for _, declared := range parameters {
		usages := references[declared.Name]
		if len(usages) == 0 {
			continue
		}
		declared.References = usages
		if value, present := currentValues[declared.Name]; present {
			declared.ExistingValue = &value
		}
		selected = append(selected, declared)
	}
	return selected
}

func collectPathParameterReferences(source composeSource) map[string][]parameter.Reference {
	pathModel := map[string]any{"include": source.model["include"]}
	pathServices := map[string]any{}
	services, _ := source.model["services"].(map[string]any)
	for name, value := range services {
		service, _ := value.(map[string]any)
		extends, _ := service["extends"].(map[string]any)
		if file, present := extends["file"]; present {
			pathServices[name] = map[string]any{"extends": map[string]any{"file": file}}
		}
	}
	pathModel["services"] = pathServices
	references := collectParameterReferences(pathModel)
	for _, usages := range references {
		for index := range usages {
			usages[index].Environment = source.environment
		}
	}
	return references
}

func decodeParameterMetadata(model map[string]any) ([]parameter.Parameter, error) {
	var document yaml.Node
	if err := document.Encode(model["x-topo"]); err != nil {
		return nil, fmt.Errorf("failed to encode project metadata: %w", err)
	}

	var raw parameterMetadata
	if err := document.Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode project metadata: %w", err)
	}

	if len(raw.Parameters) == 0 && len(raw.Args) > 0 {
		logger.Warn("x-topo.args is deprecated; use x-topo.parameters instead")
		raw.Parameters = raw.Args
	}

	parameters := make([]parameter.Parameter, 0, len(raw.Parameters))
	for name, param := range raw.Parameters {
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
