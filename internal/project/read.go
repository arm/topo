package project

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/template"
	"github.com/compose-spec/compose-go/v2/types"
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

func ReferencedEnvVars(composeFilePath string) ([]string, error) {
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
	model, err := options.LoadModel(context.Background())
	if err != nil {
		return nil, err
	}
	return slices.Collect(maps.Keys(template.ExtractVariables(model, nil))), nil
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
