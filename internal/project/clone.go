package project

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
)

func Clone(output io.Writer, path string, src Source, resolver parameter.Resolver) error {
	if err := term.PrintHeader(output, "Copy files"); err != nil {
		return err
	}
	if err := copyProject(src, path); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Configure project"); err != nil {
		return err
	}
	if err := configure(path, resolver); err != nil {
		return err
	}

	if err := term.PrintHeader(output, "Project ready"); err != nil {
		return err
	}
	return printSummary(output, path)
}

func copyProject(src Source, path string) error {
	if err := src.CopyTo(path); err != nil {
		if errDestDirExists, ok := errors.AsType[DestDirExistsError](err); ok {
			return fmt.Errorf("%w: please choose a different project directory or remove the existing directory", errDestDirExists)
		}
		return fmt.Errorf("failed to copy project: %w", err)
	}
	return nil
}

func configure(path string, resolver parameter.Resolver) error {
	composeFile, err := compose.FindDefaultFile(path)
	if err != nil {
		return err
	}
	if err := Configure(composeFile, resolver); err != nil {
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return fmt.Errorf("init failed: %w", err)
	}
	return nil
}

func printSummary(output io.Writer, path string) error {
	toPrint := fmt.Sprintf(`Created in '%s'

Now run:
  cd %s
  topo deploy

A deployment target is required. Provide --target or set TOPO_TARGET.`, path, path)

	_, err := fmt.Fprintln(output, toPrint)
	return err
}
