package compose

import (
	"context"
	"fmt"
	"sort"

	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
)

func ImageNames(composeFilePath string) ([]string, error) {
	project, err := ReadProject(composeFilePath)
	if err != nil {
		return nil, err
	}
	var names []string
	for name, svc := range project.Services {
		if svc.Image != "" {
			names = append(names, svc.Image)
		} else {
			names = append(names, fmt.Sprintf("%s-%s", project.Name, name))
		}
	}
	sort.Strings(names)
	return names, nil
}

func PullableServices(composeFilePath string) ([]string, error) {
	project, err := ReadProject(composeFilePath)
	if err != nil {
		return nil, err
	}
	var names []string
	for name, svc := range project.Services {
		if svc.Build == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func ReadProject(targetProjectFile string) (*types.Project, error) {
	return readProject(targetProjectFile, nil)
}

func ReadProjectWithEnv(targetProjectFile string, env []string) (*types.Project, error) {
	return readProject(targetProjectFile, env)
}

func readProject(targetProjectFile string, env []string) (*types.Project, error) {
	ctx := context.Background()
	options, err := cli.NewProjectOptions(
		[]string{targetProjectFile},
		cli.WithResolvedPaths(false),
		cli.WithNormalization(false),
		cli.WithEnvFiles(),
		cli.WithEnv(env),
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
	project, err := options.LoadProject(ctx)
	if err != nil {
		return nil, err
	}
	return project, nil
}
