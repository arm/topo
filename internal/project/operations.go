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

	return compose.ApplyArgs(composeFilePath, argsToMap(resolvedArgs))
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
	resolvedParameters, err := Resolve(p, argProvider)
	if err != nil {
		return nil, err
	}

	return resolvedParameters, nil
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
