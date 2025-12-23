package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/compose"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func Clone(path string, src template.Source, argProvider *arguments.StrictProviderChain, logOutput io.Writer) error {
	if err := src.CopyTo(path); err != nil {
		var errDestDirExists template.DestDirExistsError
		if errors.As(err, &errDestDirExists) {
			return fmt.Errorf("%w: please choose a different project directory or remove the existing directory", errDestDirExists)
		}
		return fmt.Errorf("failed to copy Service Template: %w", err)
	}

	composeFile := filepath.Join(path, template.ComposeFilename)
	if err := ResolveAndApplyArgs(composeFile, argProvider, logOutput); err != nil {
		if rmErr := os.RemoveAll(path); rmErr != nil {
			return errors.Join(err, rmErr)
		}
		return fmt.Errorf("init failed: %w", err)
	}

	return nil
}

func ResolveAndApplyArgs(composeFilePath string, argProvider arguments.Provider, logOutput io.Writer) error {
	resolvedArgs, err := resolveArgs(composeFilePath, argProvider)
	if err != nil {
		return fmt.Errorf("failed to resolve args: %w", err)
	}

	if len(resolvedArgs) == 0 {
		return nil
	}

	return applyArgs(composeFilePath, resolvedArgs, logOutput)
}

func Extend(targetComposeFile string, src template.Source, argProvider arguments.Provider) error {
	project, err := compose.Read(targetComposeFile)
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

	tpl, err := template.FromDir(destDir)
	if err != nil {
		return fmt.Errorf("failed to load topo template from %s: %w", src.String(), err)
	}
	if len(tpl.Services) == 0 {
		return fmt.Errorf("template found in directory %s, has no services", destDir)
	}

	resolvedTemplate, err := template.Resolve(tpl, argProvider)
	if err != nil {
		return err
	}

	extendedComposeFilePath := filepath.Join(copiedDirName, template.ComposeFilename)
	for _, service := range resolvedTemplate.Services {
		newSvc := compose.CreateServiceByExtension(extendedComposeFilePath, service.Name, argsToMap(resolvedTemplate.Args))

		if err := compose.InsertService(project, newSvc); err != nil {
			return err
		}
	}

	var allServicesVolumes []types.ServiceVolumeConfig
	for _, service := range resolvedTemplate.Services {
		volumes, err := compose.ExtractNamedServiceVolumes(service.Data)
		if err != nil {
			return err
		}
		allServicesVolumes = append(allServicesVolumes, volumes...)
	}
	compose.RegisterVolumes(project, allServicesVolumes)

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
	project, err := compose.Read(composeFilePath)
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

func applyArgs(composeFilePath string, args []arguments.ResolvedArg, logOutput io.Writer) error {
	f, err := os.Open(composeFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	yamlNodes, err := compose.ReadNodes(f)
	if err != nil {
		return err
	}

	if err := compose.ApplyArgs(yamlNodes, argsToMap(args), logOutput); err != nil {
		return fmt.Errorf("error applying args to project file: %w", err)
	}

	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(yamlNodes); err != nil {
		return err
	}
	_ = enc.Close()
	if err := os.WriteFile(composeFilePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write compose file: %s %w", composeFilePath, err)
	}
	return nil
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
