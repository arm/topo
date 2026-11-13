package project

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/override"
	"github.com/compose-spec/compose-go/v2/template"
	"github.com/compose-spec/compose-go/v2/types"
	"gopkg.in/yaml.v3"
)

func ImageNames(scope Scope) ([]string, error) {
	composeProject, err := Read(scope)
	if err != nil {
		return nil, err
	}
	var names []string
	for name, svc := range composeProject.Services {
		if svc.Image != "" {
			names = append(names, svc.Image)
		} else {
			names = append(names, fmt.Sprintf("%s-%s", composeProject.Name, name))
		}
	}
	sort.Strings(names)
	return names, nil
}

func PullableServices(scope Scope) ([]string, error) {
	composeProject, err := Read(scope)
	if err != nil {
		return nil, err
	}
	var names []string
	for name, svc := range composeProject.Services {
		if svc.Build == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func Read(scope Scope) (*types.Project, error) {
	ctx := context.Background()
	options, err := newProjectOptions(scope)
	if err != nil {
		return nil, err
	}
	err = cli.WithResolvedPaths(false)(options)
	if err != nil {
		return nil, err
	}
	err = cli.WithNormalization(false)(options)
	if err != nil {
		return nil, err
	}
	composeProject, err := options.LoadProject(ctx)
	if err != nil {
		return nil, err
	}
	return composeProject, nil
}

type composeSource struct {
	path                  string
	ancestors             []string
	extendsAncestors      [][2]string
	workingDirectory      string
	environment           types.Mapping
	referenceEnvironments map[string]types.Mapping
	model                 map[string]any
	inheritanceModel      map[string]any
}

func findIncludedServiceSource(source composeSource, sources []composeSource, service string) (*composeSource, error) {
	var found *composeSource
	for index := range sources {
		candidate := &sources[index]
		if !slices.Contains(candidate.ancestors, source.path) {
			continue
		}
		services, _ := candidate.model["services"].(map[string]any)
		if _, present := services[service]; !present {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("service %q is defined in both included files %q and %q", service, found.path, candidate.path)
		}
		found = candidate
	}
	return found, nil
}

func resolveSourceInheritance(source *composeSource, sources []composeSource) error {
	source.referenceEnvironments = map[string]types.Mapping{}
	for _, references := range collectParameterReferences(source.model) {
		for _, reference := range references {
			source.referenceEnvironments[reference.Path] = source.environment
		}
	}
	services, _ := source.model["services"].(map[string]any)
	visibleServices := maps.Clone(services)
	serviceOrigins := map[string]*composeSource{}
	for _, included := range sources {
		includedServices, _ := included.model["services"].(map[string]any)
		for name := range includedServices {
			if _, present := visibleServices[name]; present {
				continue
			}
			origin, err := findIncludedServiceSource(*source, sources, name)
			if err != nil {
				return err
			}
			if origin != nil && visibleServices != nil {
				visibleServices[name] = origin.model["services"].(map[string]any)[name]
				serviceOrigins[name] = origin
			}
		}
	}
	for range services {
		for name, value := range services {
			service, _ := value.(map[string]any)
			base, _ := service["extends"].(string)
			if config, ok := service["extends"].(map[string]any); ok {
				if config["file"] != nil {
					continue
				}
				base, _ = config["service"].(string)
			}
			if base == "" {
				continue
			}
			origin := source
			if included, present := serviceOrigins[base]; present {
				origin = included
			}
			prefix := "services." + base + "."
			for path, environment := range origin.referenceEnvironments {
				if !strings.HasPrefix(path, prefix) {
					continue
				}
				inheritedPath := "services." + name + "." + strings.TrimPrefix(path, prefix)
				if _, present := source.referenceEnvironments[inheritedPath]; !present {
					source.referenceEnvironments[inheritedPath] = environment
				}
			}
		}
	}
	inheritanceModel := maps.Clone(source.model)
	inheritanceModel["services"] = visibleServices
	for _, value := range services {
		service, _ := value.(map[string]any)
		if _, present := service["extends"]; !present {
			continue
		}
		model, err := resolveInheritanceModel(*source, inheritanceModel)
		if err != nil {
			return err
		}
		resolvedServices := model["services"].(map[string]any)
		for name := range resolvedServices {
			if _, local := services[name]; !local {
				delete(resolvedServices, name)
			}
		}
		source.model = model
		return nil
	}
	return nil
}

func newProjectOptions(scope Scope) (*cli.ProjectOptions, error) {
	return cli.NewProjectOptions(scope.ComposeFiles,
		cli.WithEnv(scope.Env),
		cli.WithOsEnv,
		cli.WithEnvFiles(scope.EnvFiles...),
		cli.WithDotEnv,
	)
}

func resolveInheritanceModel(source composeSource, model map[string]any) (map[string]any, error) {
	source.model = model
	services, _ := model["services"].(map[string]any)
	resolved := make(map[string]any, len(services))
	for name := range services {
		service, err := resolveServiceInheritance(source, name)
		if err != nil {
			return nil, err
		}
		resolved[name] = service
	}
	model["services"] = resolved
	return model, nil
}

func resolveServiceInheritance(source composeSource, service string) (map[string]any, error) {
	chain, err := collectServiceInheritance(source, service)
	if err != nil {
		return nil, err
	}
	var document yaml.Node
	if err := document.Encode(chain); err != nil {
		return nil, err
	}
	if err := document.Decode(&chain); err != nil {
		return nil, err
	}
	merged := map[string]any{}
	for index := len(chain) - 1; index >= 0; index-- {
		merged, err = override.ExtendService(merged, chain[index])
		if err != nil {
			return nil, err
		}
		delete(merged, "extends")
	}
	return merged, nil
}

func collectServiceInheritance(source composeSource, service string) ([]map[string]any, error) {
	config := types.ExtendsConfig{Service: service}
	visited := map[[2]string]bool{}
	var chain []map[string]any
	for {
		extended, err := readExtendedService(&source, config.File, config.Service)
		if err != nil {
			return nil, err
		}
		identity := [2]string{source.path, config.Service}
		if visited[identity] {
			return nil, fmt.Errorf("extends cycle detected at service %q in %q", config.Service, source.path)
		}
		visited[identity] = true
		chain = append(chain, extended)
		value, present := extended["extends"]
		if !present {
			return chain, nil
		}
		config, err = decodeExtends(value)
		if err != nil {
			return nil, err
		}
	}
}

func decodeExtends(value any) (types.ExtendsConfig, error) {
	if service, ok := value.(string); ok {
		value = map[string]any{"service": service}
	}
	var config types.ExtendsConfig
	if err := loader.Transform(value, &config); err != nil {
		return config, fmt.Errorf("decode extends: %w", err)
	}
	if config.Service == "" {
		return config, fmt.Errorf("extends must specify a service")
	}
	return config, nil
}

func readExtendedService(source *composeSource, file, service string) (map[string]any, error) {
	model := source.model
	path := source.path
	if file != "" {
		resolved, err := resolveComposeFilePath(file, source.workingDirectory, source.environment)
		if err != nil {
			return nil, err
		}
		path = resolved
		model, err = readRawComposeFile(path)
		if err != nil {
			return nil, err
		}
	}
	services, ok := model["services"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("compose file %q must contain a services mapping", path)
	}
	extended, ok := services[service].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("extends service %q in %q is missing or is not a mapping", service, path)
	}
	if file != "" {
		source.path = path
		source.model = model
		source.workingDirectory = filepath.Dir(path)
	}
	return extended, nil
}

func readRootSources(scope Scope) ([]composeSource, error) {
	options, err := newProjectOptions(scope)
	if err != nil {
		return nil, err
	}
	for name, value := range scope.EnvOverrides {
		if value == nil {
			delete(options.Environment, name)
			continue
		}
		options.Environment[name] = *value
	}
	workingDirectory, err := options.GetWorkingDir()
	if err != nil {
		return nil, err
	}
	sources := make([]composeSource, 0, len(scope.ComposeFiles))
	for _, path := range scope.ComposeFiles {
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		model, err := readRawComposeFile(path)
		if err != nil {
			return nil, err
		}
		sources = append(sources, composeSource{
			path: path, workingDirectory: workingDirectory,
			environment: options.Environment, model: model,
		})
	}
	return sources, nil
}

func discoverIncludeSources(sources []composeSource) ([]composeSource, error) {
	for index := 0; index < len(sources); index++ {
		children, err := readDirectIncludes(sources[index])
		if err != nil {
			return nil, err
		}
		sources = append(sources, children...)
	}
	return sources, nil
}

func readDirectExtendedServices(source composeSource, discovered []composeSource) ([]composeSource, error) {
	services, _ := source.model["services"].(map[string]any)
	var sources []composeSource
	for name, value := range services {
		service, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("compose file %q must contain service mappings", source.path)
		}
		extends, present := service["extends"]
		if !present {
			continue
		}
		config, err := decodeExtends(extends)
		if err != nil {
			return nil, err
		}
		child := source
		if source.inheritanceModel != nil {
			child.model = source.inheritanceModel
		}
		available, _ := child.model["services"].(map[string]any)
		if config.File == "" && available[config.Service] == nil {
			included, err := findIncludedServiceSource(source, discovered, config.Service)
			if err != nil {
				return nil, err
			}
			if included != nil {
				continue
			}
		}
		extended, err := readExtendedService(&child, config.File, config.Service)
		if err != nil {
			return nil, err
		}
		ancestors := append(slices.Clone(source.extendsAncestors), [2]string{source.path, name})
		if slices.Contains(ancestors, [2]string{child.path, config.Service}) {
			return nil, fmt.Errorf("extends cycle detected at service %q in %q", config.Service, child.path)
		}
		child.extendsAncestors = ancestors
		child.inheritanceModel = child.model
		child.model = map[string]any{"services": map[string]any{config.Service: extended}}
		sources = append(sources, child)
	}
	return sources, nil
}

func readDirectIncludes(source composeSource) ([]composeSource, error) {
	includes, err := resolveIncludes(source.model["include"], source.workingDirectory, source.environment)
	if err != nil {
		return nil, err
	}
	var sources []composeSource
	for _, config := range includes {
		for _, path := range config.Path {
			if path == source.path || slices.Contains(source.ancestors, path) {
				return nil, fmt.Errorf("include cycle detected: %q includes %q", source.path, path)
			}
		}
		children, err := readIncludeSources(config, source.environment)
		if err != nil {
			return nil, err
		}
		for childIndex := range children {
			children[childIndex].ancestors = append(slices.Clone(source.ancestors), source.path)
		}
		sources = append(sources, children...)
	}
	return sources, nil
}

func readIncludeSources(config types.IncludeConfig, parent types.Mapping) ([]composeSource, error) {
	environment, err := loadIncludeEnvironment(config, parent)
	if err != nil {
		return nil, err
	}
	sources := make([]composeSource, 0, len(config.Path))
	for _, path := range config.Path {
		model, err := readRawComposeFile(path)
		if err != nil {
			return nil, err
		}
		sources = append(sources, composeSource{
			path:             path,
			workingDirectory: config.ProjectDirectory,
			environment:      environment,
			model:            model,
		})
	}
	return sources, nil
}

func loadIncludeEnvironment(config types.IncludeConfig, parent types.Mapping) (types.Mapping, error) {
	files := config.EnvFile
	if len(files) == 0 {
		path := filepath.Join(config.ProjectDirectory, ".env")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			files = []string{path}
		}
	}
	var paths []string
	for _, path := range files {
		if path != "/dev/null" {
			paths = append(paths, path)
		}
	}
	defaults, err := dotenv.GetEnvFromFile(parent, paths)
	if err != nil {
		return nil, err
	}
	return parent.Clone().Merge(defaults), nil
}

func resolveIncludes(value any, workingDirectory string, environment map[string]string) ([]types.IncludeConfig, error) {
	if value == nil {
		return nil, nil
	}
	entries, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("include must be a list")
	}
	configs := make([]types.IncludeConfig, 0, len(entries))
	for _, entry := range entries {
		if path, ok := entry.(string); ok {
			entry = map[string]any{"path": path}
		}
		var config types.IncludeConfig
		if err := loader.Transform(entry, &config); err != nil {
			return nil, fmt.Errorf("decode include: %w", err)
		}
		if len(config.Path) == 0 {
			return nil, fmt.Errorf("include must specify a path")
		}
		for index, expression := range config.Path {
			path, err := resolveComposeFilePath(expression, workingDirectory, environment)
			if err != nil {
				return nil, err
			}
			config.Path[index] = path
		}
		if config.ProjectDirectory == "" {
			config.ProjectDirectory = filepath.Dir(config.Path[0])
		} else {
			directory, err := resolveComposeFilePath(config.ProjectDirectory, workingDirectory, environment)
			if err != nil {
				return nil, err
			}
			config.ProjectDirectory = directory
		}
		for index, expression := range config.EnvFile {
			path, err := resolveComposeFilePath(expression, workingDirectory, environment)
			if err != nil {
				return nil, err
			}
			config.EnvFile[index] = path
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func resolveComposeFilePath(expression, workingDirectory string, environment map[string]string) (string, error) {
	path, err := template.Substitute(expression, func(name string) (string, bool) {
		value, present := environment[name]
		return value, present
	})
	if err != nil {
		return "", fmt.Errorf("resolve compose file path %q: %w", expression, err)
	}
	if path == "" {
		return "", fmt.Errorf("compose file path %q resolves to an empty string", expression)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDirectory, path)
	}
	return path, nil
}

func readRawComposeFile(path string) (map[string]any, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := yaml.Unmarshal(contents, &document); err != nil {
		return nil, fmt.Errorf("parse compose file %q: %w", path, err)
	}
	return document, nil
}
