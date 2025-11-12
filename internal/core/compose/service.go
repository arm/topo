package compose

import (
	"fmt"

	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/transform"
	"github.com/compose-spec/compose-go/v2/types"
)

func ParseServiceFromTopo(serviceName string, topoService *service.TemplateManifest) (types.ServiceConfig, error) {
	// Create an in-memory compose file to dump the service definition into
	composeDict := map[string]any{
		"services": map[string]any{
			serviceName: topoService.Service,
		},
	}

	// Use compose-spec's transform.Canonical to convert the supported syntaxes to their canonical representation
	// This avoids us having to handle parsing of the various short forms
	canonical, err := transform.Canonical(composeDict, false)
	if err != nil {
		return types.ServiceConfig{}, fmt.Errorf("failed to canonicalize service config: %w", err)
	}

	servicesDict, ok := canonical["services"].(map[string]any)
	if !ok {
		return types.ServiceConfig{}, fmt.Errorf("unexpected services format")
	}

	serviceDict, ok := servicesDict[serviceName]
	if !ok {
		return types.ServiceConfig{}, fmt.Errorf("service %q not found after canonicalization", serviceName)
	}

	var svc types.ServiceConfig
	if err := loader.Transform(serviceDict, &svc); err != nil {
		return types.ServiceConfig{}, fmt.Errorf("failed to transform service config: %w", err)
	}

	svc.Name = serviceName
	svc.Build = &types.BuildConfig{Context: "./" + serviceName}

	return svc, nil
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

func RegisterNamedVolumes(targetProject *types.Project, newService types.ServiceConfig) {
	if targetProject.Volumes == nil {
		targetProject.Volumes = make(types.Volumes)
	}

	for _, vol := range newService.Volumes {
		if vol.Type == types.VolumeTypeVolume && vol.Source != "" {
			if _, exists := targetProject.Volumes[vol.Source]; !exists {
				targetProject.Volumes[vol.Source] = types.VolumeConfig{}
			}
		}
	}
}
