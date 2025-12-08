package compose

import (
	"fmt"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/transform"
	"github.com/compose-spec/compose-go/v2/types"
)

func ExtractNamedServiceVolumes(serviceName string, resolved template.ResolvedTemplate) ([]types.ServiceVolumeConfig, error) {
	// Create an in-memory compose file to dump the service definition into
	composeDict := map[string]any{
		"services": map[string]any{
			serviceName: resolved.Service,
		},
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

	serviceDict, ok := servicesDict[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %q not found after canonicalization", serviceName)
	}

	var svc types.ServiceConfig
	if err := loader.Transform(serviceDict, &svc); err != nil {
		return nil, fmt.Errorf("failed to transform service config: %w", err)
	}

	namedVolumes := []types.ServiceVolumeConfig{}
	for _, vol := range svc.Volumes {
		if vol.Type == types.VolumeTypeVolume && vol.Source != "" {
			namedVolumes = append(namedVolumes, vol)
		}
	}

	return namedVolumes, nil
}

func CreateService(serviceName string, resolved template.ResolvedTemplate) types.ServiceConfig {
	projectService := types.ServiceConfig{}
	projectService.Name = serviceName
	projectService.Extends = &types.ExtendsConfig{
		File:    "./" + serviceName + "/" + template.ComposeFilename,
		Service: resolved.ServiceName,
	}

	if args := convertResolvedArgsToBuildArgs(resolved.Args); args != nil {
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
