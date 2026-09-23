package project

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/migrate"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/parameter"
)

var ErrParameterMigrationRequired = errors.New("this project appears to use the parameter format from Topo versions older than 14.0.0")

type CloneOptions struct {
	MigrateToEnv        bool
	OutputEnvFile       string
	ProjectReadyMessage string
}

func Clone(output io.Writer, path string, src Source, resolver parameter.Resolver, options CloneOptions) error {
	if options.OutputEnvFile == "" {
		options.OutputEnvFile = filepath.Join(path, env.DefaultFilename)
	}

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

	if options.MigrateToEnv {
		if err := term.PrintNthHeader(output, "Migrate to dotenv-based configuration"); err != nil {
			return err
		}
		if err := migrateProject(output, composeFilePath, options.OutputEnvFile); err != nil {
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
		err = Configure(composeFilePath, options.OutputEnvFile, resolver)
	}
	if err != nil {
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return fmt.Errorf("configure failed: %w", err)
	}

	if options.ProjectReadyMessage != "" {
		if err := term.PrintNthHeader(output, "Project ready"); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, options.ProjectReadyMessage)
		if err != nil {
			return err
		}
	}
	return nil
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
