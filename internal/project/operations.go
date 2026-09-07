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

func Clone(path string, src Source, provider parameter.ValueProvider) error {
	return NewClone(path, src, provider).Run(nil)
}

func NewClone(path string, src Source, provider parameter.ValueProvider) operation.Sequence {
	return operation.NewSequence(
		copyProjectOperation{
			path: path,
			src:  src,
		},
		configureOperation{
			path:     path,
			provider: provider,
		},
		printSummary{
			path: path,
		},
	)
}

func Configure(composeFilePath string, provider parameter.ValueProvider) error {
	values, err := collectValues(composeFilePath, provider)
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

func collectValues(composeFilePath string, provider parameter.ValueProvider) (parameter.Values, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	project, err := FromContent(f)
	if err != nil {
		return nil, err
	}
	return provider.Provide(toDefinitions(project.Metadata.Parameters, project.currentParameterValues))
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

type configureOperation struct {
	path     string
	provider parameter.ValueProvider
}

func (o configureOperation) Description() string {
	return "Configure project"
}

func (o configureOperation) Run(_ io.Writer) error {
	composeFile := filepath.Join(o.path, compose.DefaultFileName())
	if err := Configure(composeFile, o.provider); err != nil {
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
