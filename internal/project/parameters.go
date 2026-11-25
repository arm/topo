package project

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
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

	model, err := readUninterpolated(scope.ComposeFile)
	if err != nil {
		return nil, err
	}
	includePaths, err := readIncludePaths(scope.ComposeFile, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read include paths: %w", err)
	}
	environmentFiles, err := collectEnvFiles(includePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to collect included environment files: %w", err)
	}
	currentValues, err := env.CurrentValues(append(environmentFiles, scope.EnvFiles...))
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

func readIncludePaths(composeFilePath string, visitedSet map[string]struct{}) ([]string, error) {
	composeFilePath, err := filepath.Abs(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve compose file path: %w", err)
	}
	if visitedSet == nil {
		visitedSet = make(map[string]struct{})
	}
	if _, visited := visitedSet[composeFilePath]; visited {
		return nil, nil
	}
	visitedSet[composeFilePath] = struct{}{}

	reader, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open compose file: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	var raw map[string]any
	decoder := yaml.NewDecoder(reader)

	var includes []string
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode compose file: %w", err)
	}
	if rawIncludes, ok := raw["include"].([]any); ok {
		for _, include := range rawIncludes {
			includePath, ok := include.(string)
			if !ok {
				return nil, fmt.Errorf("unsupported include entry in %q: expected a path string", composeFilePath)
			}
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(composeFilePath), includePath)
			}
			includePath = filepath.Clean(includePath)
			if _, visited := visitedSet[includePath]; visited {
				continue
			}
			includes = append(includes, includePath)
			subIncludes, err := readIncludePaths(includePath, visitedSet)
			if err != nil {
				return nil, err
			}
			includes = append(includes, subIncludes...)
		}
	}
	return includes, nil
}

func collectEnvFiles(composeFiles []string) ([]string, error) {
	var environmentFiles []string
	for _, composeFile := range composeFiles {
		paths, err := env.ResolveFiles(filepath.Dir(composeFile), env.DefaultFilenames, true)
		if err != nil {
			return nil, err
		}
		environmentFiles = append(environmentFiles, paths...)
	}
	return environmentFiles, nil
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
