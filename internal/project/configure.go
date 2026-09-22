package project

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"github.com/compose-spec/compose-go/v2/template"
)

func UsesLiteralBuildArgConfiguration(composeFilePath string) (bool, error) {
	project, err := loadProject(composeFilePath)
	if err != nil {
		return false, err
	}

	buildArgs := make(map[string]any)
	for name, values := range project.currentParameterValues {
		buildArgs[name] = strings.Join(values, "\n")
	}
	references := template.ExtractVariables(buildArgs, nil)
	hasParameterValues := false
	for _, param := range project.Metadata.Parameters {
		if _, referenced := references[param.Name]; referenced {
			return false, nil
		}
		if len(project.currentParameterValues[param.Name]) > 0 {
			hasParameterValues = true
		}
	}
	return hasParameterValues, nil
}

func Configure(composeFilePath string, resolver parameter.Resolver) error {
	envFile := filepath.Join(filepath.Dir(composeFilePath), env.DefaultFilename)
	currentValues, err := env.ReadFile(envFile)
	if errors.Is(err, os.ErrNotExist) {
		currentValues = make(map[string]string)
	} else if err != nil {
		return fmt.Errorf("failed to load current environment values: %w", err)
	}

	definitions, err := LoadParameterDefinitions(composeFilePath, currentValues)
	if err != nil {
		return fmt.Errorf("failed to load parameter definitions: %w", err)
	}

	if len(definitions) == 0 {
		return nil
	}

	if err := warnUnreferencedParameters(composeFilePath, definitions); err != nil {
		return err
	}

	values, err := resolver.Resolve(definitions)
	if err != nil {
		return fmt.Errorf("failed to collect parameter values: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	maps.Copy(currentValues, values)
	return env.WriteFile(envFile, currentValues)
}

func MigrateToEnv(composeFilePath string) error {
	projectDir := filepath.Dir(composeFilePath)
	envFilePath := filepath.Join(projectDir, env.DefaultFilename)

	_, err := os.Stat(envFilePath)
	if !os.IsNotExist(err) {
		return fmt.Errorf("env file already exists: %s; consider removing it before migrating", envFilePath)
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
				return fmt.Errorf("parameter %s has more than one current value: %q; consider consolidating the values before migrating", param.Name, currentValues)
			}
		}

		values[param.Name] = currentValues[0]
	}

	if len(values) == 0 {
		return fmt.Errorf("no parameter values to migrate; only projects with referenced parameters can be migrated")
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

func warnUnreferencedParameters(composeFilePath string, definitions []parameter.Definition) error {
	model, err := readUninterpolated(composeFilePath)
	if err != nil {
		return err
	}
	referencedEnvVars := template.ExtractVariables(model, nil)

	unreferencedDefinitions := slices.DeleteFunc(slices.Clone(definitions), func(d parameter.Definition) bool {
		_, referenced := referencedEnvVars[d.Name]
		return referenced
	})

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
