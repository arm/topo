package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
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

	return applyParameterValuesToComposeFile(composeFilePath, values)
}

func MigrateToEnv(composeFilePath string) error {
	projectDir := filepath.Dir(composeFilePath)
	envFile := filepath.Join(projectDir, env.DefaultFilename)

	_, err := os.Stat(envFile)
	if !os.IsNotExist(err) {
		return fmt.Errorf("env file already exists: %s", envFile)
	}

	project, err := loadProject(composeFilePath)
	if err != nil {
		return err
	}

	values := map[string]string{}
	for _, param := range project.Metadata.Parameters {
		if len(project.currentParameterValues[param.Name]) == 0 {
			continue
		}
		if len(project.currentParameterValues[param.Name]) != 1 {
			return fmt.Errorf("parameter %s has more than one current value", param.Name)
		}

		values[param.Name] = project.currentParameterValues[param.Name][0]
	}

	if len(values) == 0 {
		return fmt.Errorf("no parameter values to migrate")
	}

	err = env.SaveFile(envFile, values)
	if err != nil {
		return fmt.Errorf("failed to save env file: %w", err)
	}

	references := map[string]string{}
	for k := range values {
		references[k] = fmt.Sprintf("${%s?configured via topo}", k)
	}
	err = applyParameterValuesToComposeFile(composeFilePath, references)
	if err != nil {
		deleteErr := os.Remove(envFile)
		return errors.Join(fmt.Errorf("failed to apply parameter values to compose file: %w", err), deleteErr)
	}
	return nil
}

func applyParameterValuesToComposeFile(composeFilePath string, values parameter.Values) error {
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

func loadProject(composeFilePath string) (Project, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return Project{}, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	project, err := FromContent(f)
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func collectValues(composeFilePath string, resolver parameter.Resolver) (parameter.Values, error) {
	project, err := loadProject(composeFilePath)
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
