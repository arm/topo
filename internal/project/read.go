package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/dotenv"
	"github.com/compose-spec/compose-go/v2/loader"
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
	options, err := cli.NewProjectOptions(
		[]string{scope.ComposeFile},
		cli.WithResolvedPaths(false),
		cli.WithNormalization(false),
		cli.WithEnvFiles(scope.EnvFiles...),
		cli.WithEnv(scope.Env),
	)
	if err != nil {
		return nil, err
	}
	err = cli.WithOsEnv(options)
	if err != nil {
		return nil, err
	}
	err = cli.WithDotEnv(options)
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
	path             string
	ancestors        []string
	workingDirectory string
	environment      types.Mapping
	model            map[string]any
}

func resolveSourceInheritance(source *composeSource) error {
	services, _ := source.model["services"].(map[string]any)
	for _, value := range services {
		service, _ := value.(map[string]any)
		if _, present := service["extends"]; !present {
			continue
		}
		model, err := loader.LoadModelWithContext(context.Background(), types.ConfigDetails{
			WorkingDir:  source.workingDirectory,
			Environment: source.environment,
			ConfigFiles: []types.ConfigFile{{Filename: source.path, Config: source.model}},
		}, func(options *loader.Options) {
			options.SkipInterpolation = true
			options.SkipInclude = true
			options.SkipNormalization = true
			options.SkipValidation = true
			options.SkipConsistencyCheck = true
			options.ResolvePaths = false
		})
		if err != nil {
			return err
		}
		source.model = model
		return nil
	}
	return nil
}

func readRootSource(scope Scope) (composeSource, error) {
	options, err := cli.NewProjectOptions([]string{scope.ComposeFile},
		cli.WithEnv(scope.Env), cli.WithOsEnv, cli.WithEnvFiles(scope.EnvFiles...), cli.WithDotEnv)
	if err != nil {
		return composeSource{}, err
	}
	workingDirectory, err := options.GetWorkingDir()
	if err != nil {
		return composeSource{}, err
	}
	path, err := filepath.Abs(scope.ComposeFile)
	if err != nil {
		return composeSource{}, err
	}
	model, err := readRawComposeFile(path)
	if err != nil {
		return composeSource{}, err
	}
	return composeSource{
		path: path, workingDirectory: workingDirectory,
		environment: options.Environment, model: model,
	}, nil
}

func discoverIncludeSources(sources []composeSource) ([]composeSource, error) {
	for index := 0; index < len(sources); index++ {
		source := sources[index]
		includes, err := resolveIncludes(source.model["include"], source.workingDirectory, source.environment)
		if err != nil {
			return nil, err
		}
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
