package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/operation"
	"github.com/arm/topo/internal/parameter"
)

func Clone(path string, src Source, provider parameter.Provider) error {
	return NewClone(path, src, provider).Run(nil)
}

func NewClone(path string, src Source, provider parameter.Provider) operation.Sequence {
	return operation.NewSequence(
		copyProjectOperation{
			path: path,
			src:  src,
		},
		provideParametersOperation{
			path:     path,
			provider: provider,
		},
		printSummary{
			path: path,
		},
	)
}

func ResolveAndApplyParameters(composeFilePath string, provider parameter.Provider) error {
	provided, err := resolveParameters(composeFilePath, provider)
	if err != nil {
		return fmt.Errorf("failed to resolve parameters: %w", err)
	}

	if len(provided) == 0 {
		return nil
	}

	return applyParameters(composeFilePath, provided)
}

func applyParameters(composeFilePath string, provided []parameter.Provided) error {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	yamlNodes, err := compose.ReadNode(f)
	if err != nil {
		return err
	}

	err = compose.ApplyParameters(yamlNodes, parametersToMap(provided))
	if err != nil {
		return fmt.Errorf("error applying parameters to project file: %w", err)
	}

	outFile, err := os.Create(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to open compose file for writing: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	if err := compose.WriteNode(yamlNodes, outFile); err != nil {
		return fmt.Errorf("failed to write compose file after applying parameters: %w", err)
	}
	return nil
}

func resolveParameters(composeFilePath string, provider parameter.Provider) ([]parameter.Provided, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	project, err := FromContent(f)
	if err != nil {
		return nil, err
	}
	provided, err := provider.Provide(toDefinitions(project.Metadata.Parameters, project.currentParameterValues))
	if err != nil {
		return nil, err
	}
	return provided, nil
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

func parametersToMap(provided []parameter.Provided) map[string]string {
	result := map[string]string{}
	for _, entry := range provided {
		result[entry.Name] = entry.Value
	}
	return result
}

type copyProjectOperation struct {
	path string
	src  Source
}

func (o copyProjectOperation) Description() string {
	return "Copy files"
}

func (o copyProjectOperation) Run(_ io.Writer) error {
	if err := o.src.CopyTo(o.path); err != nil {
		if errDestDirExists, ok := errors.AsType[DestDirExistsError](err); ok {
			return fmt.Errorf("%w: please choose a different project directory or remove the existing directory", errDestDirExists)
		}
		return fmt.Errorf("failed to copy project: %w", err)
	}
	return nil
}

type provideParametersOperation struct {
	path     string
	provider parameter.Provider
}

func (o provideParametersOperation) Description() string {
	return "Configure project"
}

func (o provideParametersOperation) Run(_ io.Writer) error {
	composeFile := filepath.Join(o.path, ComposeFilename)
	if err := ResolveAndApplyParameters(composeFile, o.provider); err != nil {
		if rmErr := os.RemoveAll(o.path); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return fmt.Errorf("init failed: %w", err)
	}
	return nil
}

type printSummary struct {
	path string
}

func (o printSummary) Description() string {
	return "Project ready"
}

func (o printSummary) Run(w io.Writer) error {
	if w == nil {
		return nil
	}
	toPrint := fmt.Sprintf(`Created in '%s'

Now run:
  cd %s
  topo deploy

A deployment target is required. Provide --target or set TOPO_TARGET.`, o.path, o.path)

	_, err := fmt.Fprintln(w, toPrint)
	return err
}
