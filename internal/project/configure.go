package project

import (
	"fmt"
	"os"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/parameter"
)

func Configure(composeFilePath string, resolver parameter.Resolver) error {
	values, err := collectValues(composeFilePath, resolver)
	if err != nil {
		return fmt.Errorf("failed to collect parameter values: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	return applyParameterValues(composeFilePath, values)
}

func applyParameterValues(composeFilePath string, values parameter.Values) error {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	yamlNodes, err := compose.ReadNode(f)
	if err != nil {
		return err
	}

	err = compose.ApplyParameterValues(yamlNodes, values)
	if err != nil {
		return fmt.Errorf("error applying parameter values to project file: %w", err)
	}

	outFile, err := os.Create(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to open compose file for writing: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	if err := compose.WriteNode(yamlNodes, outFile); err != nil {
		return fmt.Errorf("failed to write compose file after applying parameter values: %w", err)
	}
	return nil
}

func collectValues(composeFilePath string, resolver parameter.Resolver) (parameter.Values, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	project, err := FromContent(f)
	if err != nil {
		return nil, err
	}
	return resolver.Resolve(toDefinitions(project.Metadata.Parameters, project.currentParameterValues))
}

func toDefinitions(parameters []Parameter, currentValues map[string][]string) []parameter.Definition {
	definitions := make([]parameter.Definition, len(parameters))
	for i, definition := range parameters {
		definitions[i] = parameter.Definition{
			Name:          definition.Name,
			Description:   definition.Description,
			Required:      definition.Required,
			Example:       definition.Example,
			CurrentValues: currentValues[definition.Name],
		}
	}
	return definitions
}
