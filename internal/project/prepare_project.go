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
	includePaths, err := readIncludePaths(composeFile, nil)
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

func readIncludePaths(composeFilePath string, visitedSet map[string]struct{}) ([]string, error) {
	composeFilePath, err := filepath.Abs(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve compose file path: %w", err)
	}
	if visitedSet == nil {
		visitedSet = make(map[string]struct{})
	}
	if _, visited := visitedSet[composeFilePath]; visited {
		return nil, nil
	}
	visitedSet[composeFilePath] = struct{}{}

	reader, err := os.Open(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open compose file: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	var raw map[string]any
	decoder := yaml.NewDecoder(reader)

	var includes []string
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode compose file: %w", err)
	}
	if rawIncludes, ok := raw["include"].([]any); ok {
		paths, err := expandIncludePaths(rawIncludes)
		if err != nil {
			return nil, fmt.Errorf("invalid includes in %q: %w", composeFilePath, err)
		}
		for _, include := range paths {
			includePath, ok := include.(string)
			if !ok {
				return nil, fmt.Errorf("unsupported include entry in %q: expected a path string", composeFilePath)
			}
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(composeFilePath), includePath)
			}
			includePath = filepath.Clean(includePath)
			if _, visited := visitedSet[includePath]; visited {
				continue
			}
			includes = append(includes, includePath)
			subIncludes, err := readIncludePaths(includePath, visitedSet)
			if err != nil {
				return nil, err
			}
			includes = append(includes, subIncludes...)
		}
	}
	return includes, nil
}

func expandIncludePaths(includes []any) ([]any, error) {
	var paths []any
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
			paths = append(paths, pathList...)
			continue
		}
		paths = append(paths, include)
	}
	return paths, nil
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
