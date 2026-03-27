package compose

import (
	"context"
	"fmt"
	"os"
	"strings"

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

func FilterResolvedBuildArgs(service map[string]any, resolvedArgs map[string]string) map[string]string {
	if len(resolvedArgs) == 0 {
		return nil
	}

	buildDefinition, hasBuild := service["build"]
	if !hasBuild {
		return nil
	}

	buildMap, ok := buildDefinition.(map[string]any)
	if !ok {
		return nil
	}

	argsDefinition, hasArgs := buildMap["args"]
	if !hasArgs {
		return nil
	}

	usedArgNames := extractBuildArgNames(argsDefinition)
	if len(usedArgNames) == 0 {
		return nil
	}

	filteredArgs := make(map[string]string)
	for _, argName := range usedArgNames {
		if value, exists := resolvedArgs[argName]; exists {
			filteredArgs[argName] = value
		}
	}

	if len(filteredArgs) == 0 {
		return nil
	}

	return filteredArgs
}

func extractBuildArgNames(argsDefinition any) []string {
	argNames := make(map[string]struct{})

	switch args := argsDefinition.(type) {
	case map[string]any:
		for argName := range args {
			argNames[argName] = struct{}{}
		}
	case []any:
		for _, entry := range args {
			argName, ok := parseBuildArgName(entry)
			if ok {
				argNames[argName] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(argNames))
	for argName := range argNames {
		result = append(result, argName)
	}

	return result
}

func parseBuildArgName(entry any) (string, bool) {
	argText, ok := entry.(string)
	if !ok {
		return "", false
	}

	argName, _, _ := strings.Cut(argText, "=")
	argName = strings.TrimSpace(argName)
	if argName == "" {
		return "", false
	}

	return argName, true
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

	if err := os.WriteFile(targetComposeFile, projectInYAML, 0o600); err != nil {
		return fmt.Errorf("failed to write compose file %s %w", targetComposeFile, err)
	}
	return nil
}
