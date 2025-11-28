package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/core/compose"
	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/arm-debug/topo-cli/internal/source"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

const ComposeFilename = "compose.project.yaml"

// Read parses compose file into a compose-go project.
func Read(targetProjectFile string) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions([]string{targetProjectFile}, cli.WithOsEnv, cli.WithDotEnv, cli.WithResolvedPaths(false), cli.WithNormalization(false))
	if err != nil {
		return nil, err
	}
	return cli.ProjectFromOptions(ctx, options)
}

func AddService(targetProjectFile, newServiceName string, src source.ServiceSource, argProvider arguments.Provider) error {
	project, err := Read(targetProjectFile)
	if err != nil {
		return fmt.Errorf("failed to read project: %w", err)
	}

	destDir := filepath.Join(filepath.Dir(targetProjectFile), newServiceName)

	if err := src.CopyTo(destDir); err != nil {
		var errDestDirExists source.DestDirExistsError
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

	serviceManifest, err := service.ParseDefinition(destDir)
	if err != nil {
		return fmt.Errorf("failed to load topo service from %s: %w", src.String(), err)
	}

	resolvedTemplate, err := service.ResolveTemplate(serviceManifest, argProvider)
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
	project, err := Read(composeFilePath)
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
	if err := os.WriteFile(composePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}
	return nil
}
