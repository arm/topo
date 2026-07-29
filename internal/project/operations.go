package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/operation"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func Clone(path string, src Source, argProvider arguments.Provider) error {
	return NewClone(path, src, argProvider).Run(nil)
}

func NewClone(path string, src Source, argProvider arguments.Provider) operation.Sequence {
	return operation.NewSequence(
		copyProjectOperation{
			path: path,
			src:  src,
		},
		resolveArgsOperation{
			path:        path,
			argProvider: argProvider,
		},
		printSummary{
			path: path,
		},
	)
}

func ResolveAndApplyArgs(composeFilePath string, argProvider arguments.Provider) error {
	resolvedArgs, err := resolveArgs(composeFilePath, argProvider)
	if err != nil {
		return fmt.Errorf("failed to resolve parameters: %w", err)
	}

	if len(resolvedArgs) == 0 {
		return nil
	}

	return applyArgs(composeFilePath, resolvedArgs)
}

func RemoveService(composeFilePath, serviceName string) error {
	fileToRead, err := os.Open(composeFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = fileToRead.Close() }()
	project, err := compose.ReadNode(fileToRead)
	if err != nil {
		return err
	}

	if err := compose.RemoveService(project, serviceName); err != nil {
		return fmt.Errorf("failed to remove service %s: %w", serviceName, err)
	}

	fileToWrite, err := os.Create(composeFilePath)
	if err != nil {
		return fmt.Errorf("failed to open compose file for writing: %w", err)
	}
	defer func() { _ = fileToWrite.Close() }()

	if err := compose.WriteNode(project, fileToWrite); err != nil {
		return fmt.Errorf("failed to write compose file after removing service: %w", err)
	}

	return nil
}

func Init(projectDir string) error {
	composePath := filepath.Join(projectDir, ComposeFilename)
	if _, err := os.Stat(composePath); err == nil {
		return fmt.Errorf("compose file already exists at %s", composePath)
	} else if !os.IsNotExist(err) {
		return err
	}
	compose := types.Project{
		Services: types.Services{},
	}
	data, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("failed to marshal compose file: %w", err)
	}
	if err := os.WriteFile(composePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	return nil
}

func applyArgs(composeFilePath string, args []arguments.ResolvedArg) error {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	yamlNodes, err := compose.ReadNode(f)
	if err != nil {
		return err
	}

	err = compose.ApplyArgs(yamlNodes, argsToMap(args))
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

func resolveArgs(composeFilePath string, argProvider arguments.Provider) ([]arguments.ResolvedArg, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	p, err := FromContent(f)
	if err != nil {
		return nil, err
	}
	resolvedProject, err := Resolve(p, argProvider)
	if err != nil {
		return nil, err
	}

	return resolvedProject.Parameters, nil
}

func argsToMap(args []arguments.ResolvedArg) map[string]string {
	result := map[string]string{}
	for _, arg := range args {
		result[arg.Name] = arg.Value
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

type resolveArgsOperation struct {
	path        string
	argProvider arguments.Provider
}

func (o resolveArgsOperation) Description() string {
	return "Configure project"
}

func (o resolveArgsOperation) Run(_ io.Writer) error {
	composeFile := filepath.Join(o.path, ComposeFilename)
	if err := ResolveAndApplyArgs(composeFile, o.argProvider); err != nil {
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
