package project

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
)

var ErrNoParameterReferences = errors.New("none of the declared parameters are referenced as environment variables in the Compose file")

func Configure(composeFilePath string, resolver parameter.Resolver) error {
	envFile := filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename)
	currentValues, err := env.ReadFile(envFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to load current environment values: %w", err)
	}

	definitions, err := LoadParameterDefinitions(composeFilePath, currentValues)
	if err != nil {
		return fmt.Errorf("failed to load parameter definitions: %w", err)
	}

	if len(definitions) == 0 {
		return nil
	}

	if err := validateParameterReferences(composeFilePath, definitions); err != nil {
		if errors.Is(err, ErrNoParameterReferences) {
			return fmt.Errorf("%w; this project might use the parameter format supported by Topo versions older than 14.0.0. Try running 'topo configure --migrate-to-env', then retry configuration", err)
		}
		return err
	}

	values, err := resolver.Resolve(definitions)
	if err != nil {
		return fmt.Errorf("failed to collect parameter values: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	if currentValues == nil {
		currentValues = make(map[string]string)
	}
	maps.Copy(currentValues, values)
	return env.WriteFile(envFile, currentValues)
}

func MigrateToEnv(composeFilePath string) error {
	projectDir := filepath.Dir(composeFilePath)
	envFilePath := filepath.Join(projectDir, env.DefaultFilename)

	_, err := os.Stat(envFilePath)
	if !os.IsNotExist(err) {
		return fmt.Errorf("env file already exists: %s", envFilePath)
	}

	project, err := loadProject(composeFilePath)
	if err != nil {
		return err
	}

	values := map[string]string{}
	for _, param := range project.Metadata.Parameters {
		currentValues := project.currentParameterValues[param.Name]
		if len(currentValues) == 0 {
			continue
		}
		for _, value := range currentValues[1:] {
			if value != currentValues[0] {
				return fmt.Errorf("parameter %s has more than one current value", param.Name)
			}
		}

		values[param.Name] = currentValues[0]
	}

	if len(values) == 0 {
		return fmt.Errorf("no parameter values to migrate")
	}

	err = env.WriteFile(envFilePath, values)
	if err != nil {
		return fmt.Errorf("failed to save env file: %w", err)
	}

	references := map[string]string{}
	for k := range values {
		references[k] = fmt.Sprintf("${%s?configured via topo}", k)
	}
	err = applyParameterValuesToComposeFile(composeFilePath, references)
	if err != nil {
		deleteErr := os.Remove(envFilePath)
		return errors.Join(fmt.Errorf("failed to apply parameter values to compose file: %w", err), deleteErr)
	}
	return nil
}

func validateParameterReferences(composeFilePath string, definitions []parameter.Definition) error {
	referencedEnvVars, err := ReferencedEnvVars(composeFilePath)
	if err != nil {
		return err
	}

	unreferencedDefinitions := slices.DeleteFunc(slices.Clone(definitions), func(d parameter.Definition) bool {
		return slices.Contains(referencedEnvVars, d.Name)
	})

	if len(unreferencedDefinitions) == len(definitions) {
		return ErrNoParameterReferences
	}

	for _, param := range unreferencedDefinitions {
		logger.Warn(fmt.Sprintf("parameter %q is not referenced through an environment variable in the Compose file; configuring it will have no effect", param.Name))
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
