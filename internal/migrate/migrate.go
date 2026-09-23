package migrate

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/env"
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

func ToEnv(composeFilePath, envFilePath string) error {
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

func loadProject(composeFilePath string) (Project, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return Project{}, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	project, err := ParseProject(f)
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func applyParameterValuesToComposeFile(composeFilePath string, values map[string]string) error {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	yamlNodes, err := ReadComposeNode(f)
	if err != nil {
		return err
	}

	err = ApplyParameterValuesToCompose(yamlNodes, values)
	if err != nil {
		return fmt.Errorf("error applying parameter values to project file: %w", err)
	}

	outFile, err := os.Create(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to open compose file for writing: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	if err := WriteComposeNode(yamlNodes, outFile); err != nil {
		return fmt.Errorf("failed to write compose file after applying parameter values: %w", err)
	}
	return nil
}
