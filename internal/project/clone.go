package project

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/migrate"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
)

var ErrParameterMigrationRequired = errors.New("this project appears to use the parameter format from Topo versions older than 14.0.0")

func Clone(output io.Writer, path string, src Source, resolver parameter.Resolver, outputEnvFilePath string, migrateToEnv bool) error {
	if err := term.PrintFirstHeader(output, "Copy files"); err != nil {
		return err
	}
	if err := copyProject(src, path); err != nil {
		return err
	}

	composeFilePath, err := compose.FindDefaultFile(path)
	if err != nil {
		return err
	}

	if migrateToEnv {
		if err := term.PrintNthHeader(output, "Migrate to dotenv-based configuration"); err != nil {
			return err
		}
		if err := migrateProject(output, composeFilePath, outputEnvFilePath); err != nil {
			if rmErr := os.RemoveAll(path); rmErr != nil {
				return errors.Join(err, rmErr)
			}
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	if err := term.PrintNthHeader(output, "Configure project"); err != nil {
		return err
	}
	usesLiteralBuildArgs, err := migrate.UsesLiteralBuildArgConfiguration(composeFilePath)
	if err == nil && usesLiteralBuildArgs {
		err = ErrParameterMigrationRequired
	}
	if err == nil {
		err = Configure(composeFilePath, outputEnvFilePath, resolver)
	}
	if err != nil {
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return fmt.Errorf("configure failed: %w", err)
	}

	if err := term.PrintNthHeader(output, "Project ready"); err != nil {
		return err
	}
	return printSummary(output, path)
}

func migrateProject(output io.Writer, composeFilePath, outputEnvFilePath string) error {
	if err := migrate.ToEnv(composeFilePath, outputEnvFilePath); err != nil {
		return err
	}
	_, err := fmt.Fprintf(output, "successfully migrated %q to be parameterized from %q\n", composeFilePath, outputEnvFilePath)
	if err != nil {
		return err
	}
	return nil
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

func printSummary(output io.Writer, path string) error {
	toPrint := fmt.Sprintf(`Created in '%s'

Now run:
  cd %s
  topo deploy

A deployment target is required. Provide --target or set TOPO_TARGET.`, path, path)

	_, err := fmt.Fprintln(output, toPrint)
	return err
}
