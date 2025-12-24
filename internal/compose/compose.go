package compose

import (
	"context"
	"fmt"
	"os"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/transform"
	"github.com/compose-spec/compose-go/v2/types"
)

func ExtractNamedServiceVolumes(service map[string]any) ([]types.ServiceVolumeConfig, error) {
	composeDict := map[string]any{
		"services": map[string]any{
			"doesnt-matter": service,
		},
	}

	canonical, err := transform.Canonical(composeDict, false)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize service config: %w", err)
	}

	namedVolumes := []types.ServiceVolumeConfig{}

	serviceDef, ok := canonical["services"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("service not found after canonicalization")
	}

	var svc types.ServiceConfig
	if err := loader.Transform(serviceDef["doesnt-matter"], &svc); err != nil {
		return nil, fmt.Errorf("failed to transform service config: %w", err)
	}

	for _, vol := range svc.Volumes {
		if vol.Type == types.VolumeTypeVolume && vol.Source != "" {
			namedVolumes = append(namedVolumes, vol)
		}
	}

	return namedVolumes, nil
}

func CreateServiceByExtension(referencedComposeFilePath string, serviceName string, args map[string]string) types.ServiceConfig {
	svc := types.ServiceConfig{}
	svc.Name = serviceName
	svc.Extends = &types.ExtendsConfig{
		File:    referencedComposeFilePath,
		Service: serviceName,
	}

	if args := convertArgs(args); args != nil {
		svc.Build = &types.BuildConfig{}
		svc.Build.Args = args
	}

	return svc
}

func convertArgs(resolvedArgs map[string]string) types.MappingWithEquals {
	if len(resolvedArgs) == 0 {
		return nil
	}

	argsSlice := make([]string, 0, len(resolvedArgs))
	for name, value := range resolvedArgs {
		argsSlice = append(argsSlice, fmt.Sprintf("%s=%s", name, value))
	}

	return types.NewMappingWithEquals(argsSlice)
}

func InsertService(p *types.Project, svc types.ServiceConfig) error {
	if p.Services == nil {
		p.Services = types.Services{}
	}
	if _, exists := p.Services[svc.Name]; exists {
		return fmt.Errorf("service %q already exists", svc.Name)
	}
	p.Services[svc.Name] = svc
	return nil
}

func RegisterVolumes(targetProject *types.Project, volumes []types.ServiceVolumeConfig) {
	if targetProject.Volumes == nil {
		targetProject.Volumes = make(types.Volumes)
	}

	for _, vol := range volumes {
		if _, exists := targetProject.Volumes[vol.Source]; !exists {
			targetProject.Volumes[vol.Source] = types.VolumeConfig{}
		}
	}
}

func ReadProject(targetProjectFile string) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions(
		[]string{targetProjectFile},
		cli.WithResolvedPaths(false),
		cli.WithNormalization(false),
	)
	if err != nil {
		return nil, err
	}
	project, err := options.LoadProject(ctx)
	if err != nil {
		return nil, err
	}
	return project, nil
}

func WriteProject(project *types.Project, targetComposeFile string) error {
	projectInYAML, err := project.MarshalYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal project to YAML: %w", err)
	}

	if err := os.WriteFile(targetComposeFile, projectInYAML, 0o644); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", targetComposeFile, err)
	}
	return nil
}
