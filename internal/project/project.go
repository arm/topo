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

func Extend(targetComposeFile string, src template.Source, argProvider arguments.Provider) error {
	project, err := parse.Read(targetComposeFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}

	absoluteTargetComposeFile, err := filepath.Abs(targetComposeFile)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of target compose file: %w", err)
	}
	currentDir := filepath.Dir(absoluteTargetComposeFile)

	originalDirName, err := src.GetName()
	if err != nil {
		return fmt.Errorf("failed to get repo name from source: %w", err)
	}

	copiedDirName := originalDirName
	for i := 1; ; i++ {
		destPath := filepath.Join(currentDir, copiedDirName)
		_, err := os.Stat(destPath)
		if err != nil {
			if os.IsNotExist(err) {
				break
			} else {
				return fmt.Errorf("failed to check if directory exists: %w", err)
			}
		}
		copiedDirName = fmt.Sprintf("%s_%d", originalDirName, i)
	}

	destDir := filepath.Join(currentDir, copiedDirName)

	var success bool
	defer func() {
		if !success {
			_ = os.RemoveAll(destDir)
		}
	}()

	if err := src.CopyTo(destDir); err != nil {
		return fmt.Errorf("failed to copy Service Template: %w", err)
	}

	if info, err := os.Stat(destDir); err != nil || !info.IsDir() {
		return fmt.Errorf("failed to find copied template directory: %w", err)
	}

	templates, err := template.ParseComposeFileToTemplate(destDir)
	if err != nil {
		return fmt.Errorf("failed to load topo template from %s: %w", src.String(), err)
	}
	if len(templates.Services) == 0 {
		return fmt.Errorf("template found in directory %s, has no services", destDir)
	}

	resolvedTemplate, err := template.Resolve(templates, argProvider)
	if err != nil {
		return err
	}

	for _, service := range resolvedTemplate.Services {
		newSvc := compose.CreateService(copiedDirName, service, resolvedTemplate.Args)

		if err := compose.InsertService(project, newSvc); err != nil {
			return err
		}
	}

	volumes, err := compose.ExtractNamedServiceVolumes(resolvedTemplate.Services)
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
	if err := os.WriteFile(targetComposeFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", targetComposeFile, err)
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
