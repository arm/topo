package compose

import (
	"fmt"
	"io"
	"sort"

	"gopkg.in/yaml.v3"
)

type composeFileSchema struct {
	Services map[string]struct {
		Build any `yaml:"build"`
	} `yaml:"services"`
}

func PullableServices(composeFile io.Reader) ([]string, error) {
	data, err := io.ReadAll(composeFile)
	if err != nil {
		return nil, err
	}
	var cf composeFileSchema
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("parsing compose file: %w", err)
	}
	var names []string
	for name, svc := range cf.Services {
		if svc.Build == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}
