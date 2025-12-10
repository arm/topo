package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/core/compose"
	"github.com/arm-debug/topo-cli/internal/project/parse"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func Clone(path string, src template.Source, argProvider *arguments.StrictProviderChain, w io.Writer) error {
	if err := src.CopyTo(path); err != nil {
		var errDestDirExists template.DestDirExistsError
		if errors.As(err, &errDestDirExists) {
			return fmt.Errorf("%w: please choose a different project directory or remove the existing directory", errDestDirExists)
		}
		return fmt.Errorf("failed to copy Service Template: %w", err)
	}

	composeFile := filepath.Join(path, template.ComposeFilename)
	if err := InitTemplate(composeFile, argProvider, w); err != nil {
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return fmt.Errorf("init failed: %w", err)
	}

	return nil
}

func InitTemplate(composeFile string, argCollector arguments.Provider, w io.Writer) error {
	proj, err := parse.ReadNodes(composeFile)
	if err != nil {
		return fmt.Errorf("error reading project file: %w", err)
	}

	requiredArgs := parse.ListArgs(proj)
	if len(requiredArgs) == 0 {
		return nil
	}

	resolvedArgs, err := argCollector.Provide(requiredArgs)
	if err != nil {
		return err
	}

	if err := parse.ApplyArgs(proj, resolvedArgs, w); err != nil {
		return fmt.Errorf("error applying args to project file: %w", err)
	}

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(proj); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(composeFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file: %s %w", composeFile, err)
	}
	return nil
}

func AddService(targetProjectFile, newServiceName string, src template.Source, argProvider arguments.Provider) error {
	project, err := parse.Read(targetProjectFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}

	destDir := filepath.Join(filepath.Dir(targetProjectFile), newServiceName)

	if err := src.CopyTo(destDir); err != nil {
		var errDestDirExists template.DestDirExistsError
		if errors.As(err, &errDestDirExists) {
			return fmt.Errorf("%w: please choose a different service name or remove the existing directory", errDestDirExists)
		}
		return fmt.Errorf("failed to copy Service Template: %w", err)
	}

	var success bool
	defer func() {
		if !success {
			_ = os.RemoveAll(destDir)
		}
	}()

	tmpl, err := template.ParseDefinition(destDir)
	if err != nil {
		return fmt.Errorf("failed to load topo template from %s: %w", src.String(), err)
	}

	resolvedTemplate, err := template.Resolve(tmpl, argProvider)
	if err != nil {
		return err
	}

	newSvc := compose.CreateService(newServiceName, resolvedTemplate)

	if err := compose.InsertService(project, newSvc); err != nil {
		return err
	}

	volumes, err := compose.ExtractNamedServiceVolumes(
		newServiceName,
		resolvedTemplate,
	)
	if err != nil {
		return err
	}
	compose.RegisterVolumes(project, volumes)

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(targetProjectFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", targetProjectFile, err)
	}

	success = true
	return nil
}

func RemoveService(composeFilePath, serviceName string) error {
	project, err := parse.Read(composeFilePath)
	if err != nil {
		return err
	}
	newServices := types.Services{}
	for k, svc := range project.Services {
		if k == serviceName {
			continue
		}
		newServices[k] = svc
	}
	project.Services = newServices
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(project); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(composeFilePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", composeFilePath, err)
	}
	return nil
}

func Init(projectDir string) error {
	composePath := filepath.Join(projectDir, template.ComposeFilename)
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
	if err := os.WriteFile(composePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	return nil
}
