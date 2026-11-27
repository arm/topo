package project

import (
	"context"
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

func ImageNames(preparedProject PreparedProject) ([]string, error) {
	composeProject, err := Read(preparedProject)
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

func PullableServices(preparedProject PreparedProject) ([]string, error) {
	composeProject, err := Read(preparedProject)
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

func Read(preparedProject PreparedProject) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions(
		[]string{preparedProject.ComposeFile()},
		cli.WithResolvedPaths(false),
		cli.WithNormalization(false),
		cli.WithEnvFiles(preparedProject.EnvFiles...),
		cli.WithEnv(preparedProject.Env),
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
