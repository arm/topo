package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/env"
)

func ToEnv(composeFilePath string) error {
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

	changes := map[string]*string{}
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

		changes[param.Name] = new(currentValues[0])
	}

	if len(changes) == 0 {
		return fmt.Errorf("no parameter values to migrate; only projects with referenced parameters can be migrated")
	}

	err = env.UpdateFile(envFilePath, changes, env.EncodeOptions{PreserveInterpolation: true})
	if err != nil {
		return fmt.Errorf("failed to save env file: %w", err)
	}

	references := map[string]string{}
	for k := range changes {
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
