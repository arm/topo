package compose

import (
	"fmt"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/transform"
	"github.com/compose-spec/compose-go/v2/types"
)

func ExtractNamedServiceVolumes(services []template.Service) ([]types.ServiceVolumeConfig, error) {
	// Create an in-memory compose file to dump the service definition into
	composeDict := map[string]any{
		"services": map[string]any{},
	}
	for _, service := range services {
		composeDict["services"].(map[string]any)[service.Name] = service.Data
	}

	// Use compose-spec's transform.Canonical to convert the supported syntaxes to their canonical representation
	// This avoids us having to handle parsing of the various short forms
	canonical, err := transform.Canonical(composeDict, false)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize service config: %w", err)
	}

	servicesDict, ok := canonical["services"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected services format")
	}

	namedVolumes := []types.ServiceVolumeConfig{}
	for _, service := range services {
		serviceDict, ok := servicesDict[service.Name]
		if !ok {
			return nil, fmt.Errorf("service %q not found after canonicalization", service.Name)
		}

		var svc types.ServiceConfig
		if err := loader.Transform(serviceDict, &svc); err != nil {
			return nil, fmt.Errorf("failed to transform service config: %w", err)
		}

		for _, vol := range svc.Volumes {
			if vol.Type == types.VolumeTypeVolume && vol.Source != "" {
				namedVolumes = append(namedVolumes, vol)
			}
		}
	}

	return namedVolumes, nil
}

func CreateService(templateRepoPath string, service template.Service, args []arguments.ResolvedArg) types.ServiceConfig {
	projectService := types.ServiceConfig{}
	projectService.Name = service.Name
	projectService.Extends = &types.ExtendsConfig{
		File:    "./" + templateRepoPath + "/" + template.ComposeFilename,
		Service: service.Name,
	}

	if args := convertResolvedArgsToBuildArgs(args); args != nil {
		projectService.Build = &types.BuildConfig{}
		projectService.Build.Args = args
	}

	return projectService
}

func convertResolvedArgsToBuildArgs(resolvedArgs []arguments.ResolvedArg) types.MappingWithEquals {
	if len(resolvedArgs) == 0 {
		return nil
	}

	argsSlice := make([]string, 0, len(resolvedArgs))
	for _, arg := range resolvedArgs {
		argsSlice = append(argsSlice, fmt.Sprintf("%s=%s", arg.Name, arg.Value))
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
