package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/template"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func Clone(path string, src template.Source, argProvider arguments.Provider) ([]logger.Entry, error) {
	if err := src.CopyTo(path); err != nil {
		var errDestDirExists template.DestDirExistsError
		if errors.As(err, &errDestDirExists) {
			return nil, fmt.Errorf("%w: please choose a different project directory or remove the existing directory", errDestDirExists)
		}
		return nil, fmt.Errorf("failed to copy Service Template: %w", err)
	}

	composeFile := filepath.Join(path, template.ComposeFilename)

	logs, err := ResolveAndApplyArgs(composeFile, argProvider)
	if err != nil {
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return logs, errors.Join(err, rmErr)
		}
		return logs, fmt.Errorf("init failed: %w", err)
	}

	return logs, nil
}

func ResolveAndApplyArgs(composeFilePath string, argProvider arguments.Provider) ([]logger.Entry, error) {
	resolvedArgs, err := resolveArgs(composeFilePath, argProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve args: %w", err)
	}

	if len(resolvedArgs) == 0 {
		return nil, nil
	}

	return applyArgs(composeFilePath, resolvedArgs)
}

func Extend(targetComposeFile string, src template.Source, argProvider arguments.Provider) ([]logger.Entry, error) {
	project, err := compose.ReadProject(targetComposeFile)
	logs := []logger.Entry{}
	logs = append(logs, logger.Entry{
		Level:   logger.Warning,
		Message: "reading project compose file",
	})
	if err != nil {
		return logs, fmt.Errorf("failed to read project: %w", err)
	}

	absoluteTargetComposeFile, err := filepath.Abs(targetComposeFile)
	if err != nil {
		return logs, fmt.Errorf("failed to get absolute path of target compose file: %w", err)
	}
	currentDir := filepath.Dir(absoluteTargetComposeFile)

	originalDirName, err := src.GetName()
	if err != nil {
		return logs, fmt.Errorf("failed to get repo name from source: %w", err)
	}

	copiedDirName := originalDirName
	for i := 1; ; i++ {
		destPath := filepath.Join(currentDir, copiedDirName)
		_, err := os.Stat(destPath)
		if err != nil {
			if os.IsNotExist(err) {
				break
			} else {
				return logs, fmt.Errorf("failed to check if directory exists: %w", err)
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

	logs = append(logs, logger.Entry{
		Level:   logger.Info,
		Message: fmt.Sprintf("copying service template to %q", destDir),
	})

	if err := src.CopyTo(destDir); err != nil {
		return logs, fmt.Errorf("failed to copy Service Template: %w", err)
	}

	if info, err := os.Stat(destDir); err != nil || !info.IsDir() {
		return logs, fmt.Errorf("failed to find copied template directory: %w", err)
	}

	tpl, err := template.FromDir(destDir)
	if err != nil {
		return logs, fmt.Errorf("failed to load topo template from %s: %w", src.String(), err)
	}
	if len(tpl.Services) == 0 {
		return logs, fmt.Errorf("template found in directory %s, has no services", destDir)
	}

	resolvedTemplate, err := template.Resolve(tpl, argProvider)
	if err != nil {
		return logs, err
	}

	extendedComposeFilePath := filepath.Join(copiedDirName, template.ComposeFilename)
	for _, service := range resolvedTemplate.Services {
		newSvc := compose.CreateServiceByExtension(extendedComposeFilePath, service.Name, argsToMap(resolvedTemplate.Args))
		logs = append(logs, logger.Entry{
			Level:   logger.Info,
			Message: fmt.Sprintf("adding service %q to project", newSvc.Name),
		})
		if err := compose.InsertService(project, newSvc); err != nil {
			return logs, err
		}
	}

	var allServicesVolumes []types.ServiceVolumeConfig
	for _, service := range resolvedTemplate.Services {
		volumes, err := compose.ExtractNamedServiceVolumes(service.Data)
		if err != nil {
			return logs, err
		}
		allServicesVolumes = append(allServicesVolumes, volumes...)
	}
	compose.RegisterVolumes(project, allServicesVolumes)

	err = compose.WriteProject(project, targetComposeFile)
	if err != nil {
		return logs, err
	}

	success = true
	logs = append(logs, logger.Entry{
		Level:   logger.Info,
		Message: "successfully extended project",
	})
	return logs, nil
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

func applyArgs(composeFilePath string, args []arguments.ResolvedArg) ([]logger.Entry, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	yamlNodes, err := compose.ReadNode(f)
	if err != nil {
		return nil, err
	}

	logs, err := compose.ApplyArgs(yamlNodes, argsToMap(args))
	if err != nil {
		return logs, fmt.Errorf("error applying args to project file: %w", err)
	}

	outFile, err := os.Create(composeFilePath)
	if err != nil {
		return logs, fmt.Errorf("failed to open compose file for writing: %w", err)
	}
	defer func() { _ = outFile.Close() }()

	if err := compose.WriteNode(yamlNodes, outFile); err != nil {
		return logs, fmt.Errorf("failed to write compose file after applying args: %w", err)
	}
	return logs, nil
}

func resolveArgs(composeFilePath string, argProvider arguments.Provider) ([]arguments.ResolvedArg, error) {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("can't read compose file: %w", err)
	}
	defer func() { _ = f.Close() }()

	tpl, err := template.FromContent(f)
	if err != nil {
		return nil, err
	}
	resolvedTpl, err := template.Resolve(tpl, argProvider)
	if err != nil {
		return nil, err
	}

	return resolvedTpl.Args, nil
}

func argsToMap(args []arguments.ResolvedArg) map[string]string {
	result := map[string]string{}
	for _, arg := range args {
		result[arg.Name] = arg.Value
	}
	return result
}
