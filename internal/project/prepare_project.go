package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arm/topo/internal/env"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	"gopkg.in/yaml.v3"
)

type PreparedProject struct {
	composeFile string
	Env         []string
	EnvFiles    []string

	uninterpolatedComposeConfig map[string]any
}

func (p PreparedProject) ComposeFile() string {
	return p.composeFile
}

func PrepareProject(composeFile string, target string, envFiles []string) (PreparedProject, error) {
	targetEnv := []string{}
	if target != "" {
		var err error
		targetEnv, err = env.ResolveTargetEnv(target)
		if err != nil {
			return PreparedProject{}, err
		}
	}

	model, err := readUninterpolated(composeFile)
	if err != nil {
		return PreparedProject{}, err
	}
	includePaths, err := collectIncludePaths(composeFile)
	if err != nil {
		return PreparedProject{}, fmt.Errorf("failed to read include paths: %w", err)
	}
	environmentFiles, err := collectEnvFiles(includePaths)
	if err != nil {
		return PreparedProject{}, fmt.Errorf("failed to collect included environment files: %w", err)
	}
	return PreparedProject{
		composeFile:                 composeFile,
		Env:                         targetEnv,
		EnvFiles:                    append(environmentFiles, envFiles...),
		uninterpolatedComposeConfig: model,
	}, nil
}

func collectIncludePaths(root string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve compose file path: %w", err)
	}
	pending := []composeSource{{path: root, projectDirectory: filepath.Dir(root)}}
	visited := make(map[string]struct{})
	var paths []string
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		if _, seen := visited[current.path]; seen {
			continue
		}
		visited[current.path] = struct{}{}
		children, err := readIncludedSources(current)
		if err != nil {
			return nil, err
		}
		if current.path != root {
			paths = append(paths, current.path)
		}
		pending = append(children, pending...)
	}
	return paths, nil
}

func readComposeConfig(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file %q: %w", path, err)
	}
	var config map[string]any
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to decode compose file %q: %w", path, err)
	}
	return config, nil
}

type composeSource struct {
	path             string
	projectDirectory string
}

func readIncludedSources(source composeSource) ([]composeSource, error) {
	config, err := readComposeConfig(source.path)
	if err != nil {
		return nil, err
	}
	includes, _ := config["include"].([]any)
	groups, err := collectIncludePathGroups(includes)
	if err != nil {
		return nil, fmt.Errorf("invalid includes in %q: %w", source.path, err)
	}
	var sources []composeSource
	for _, group := range groups {
		groupSources, err := resolveIncludeGroup(group, source.projectDirectory)
		if err != nil {
			return nil, fmt.Errorf("invalid include group in %q: %w", source.path, err)
		}
		sources = append(sources, groupSources...)
	}
	return sources, nil
}

func resolveIncludeGroup(paths []any, projectDirectory string) ([]composeSource, error) {
	var sources []composeSource
	groupDirectory := ""
	for _, path := range paths {
		includePath, ok := path.(string)
		if !ok {
			return nil, fmt.Errorf("unsupported include entry: expected a path string")
		}
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(projectDirectory, includePath)
		}
		includePath = filepath.Clean(includePath)
		if groupDirectory == "" {
			groupDirectory = filepath.Dir(includePath)
		}
		sources = append(sources, composeSource{path: includePath, projectDirectory: groupDirectory})
	}
	return sources, nil
}

func collectIncludePathGroups(includes []any) ([][]any, error) {
	var groups [][]any
	for _, include := range includes {
		if config, ok := include.(map[string]any); ok {
			for _, field := range []string{"project_directory", "env_file"} {
				if _, present := config[field]; present {
					return nil, fmt.Errorf("unsupported include field %q", field)
				}
			}
			include = config["path"]
		}
		if pathList, ok := include.([]any); ok {
			groups = append(groups, pathList)
			continue
		}
		groups = append(groups, []any{include})
	}
	return groups, nil
}

func collectEnvFiles(composeFiles []string) ([]string, error) {
	var environmentFiles []string
	for _, composeFile := range composeFiles {
		paths, err := env.ResolveFiles(filepath.Dir(composeFile), env.DefaultFilenames, true)
		if err != nil {
			return nil, err
		}
		environmentFiles = append(environmentFiles, paths...)
	}
	return environmentFiles, nil
}

func readUninterpolated(composeFilePath string) (map[string]any, error) {
	options, err := cli.NewProjectOptions([]string{composeFilePath},
		cli.WithResolvedPaths(false),
		cli.WithNormalization(false),
		cli.WithLoadOptions(func(o *loader.Options) {
			o.SkipInterpolation = true
		}),
	)
	if err != nil {
		return nil, err
	}
	return options.LoadModel(context.Background())
}
