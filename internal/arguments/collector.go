package arguments

import (
	"fmt"
	"maps"

	"github.com/arm-debug/topo-cli/internal/service"
)

type Collector []Provider

func NewCollector(providers ...Provider) Collector {
	return Collector(providers)
}

func (c Collector) Collect(specs []service.ArgSpec) (map[string]string, error) {
	result := make(map[string]string)

	for _, provider := range c {
		args, err := provider.Provide(specs)
		if err != nil {
			return nil, fmt.Errorf("%s provider failed: %w", provider.Name(), err)
		}
		maps.Copy(result, args)

		if allRequiredProvided(specs, result) {
			break
		}
	}

	if err := validateRequired(specs, result); err != nil {
		return nil, err
	}

	return result, nil
}

func allRequiredProvided(specs []service.ArgSpec, provided map[string]string) bool {
	for _, spec := range specs {
		if spec.Required {
			if value, exists := provided[spec.Name]; !exists || value == "" {
				return false
			}
		}
	}
	return true
}

func validateRequired(specs []service.ArgSpec, provided map[string]string) error {
	var missing []service.ArgSpec
	for _, spec := range specs {
		if spec.Required {
			if value, exists := provided[spec.Name]; !exists || value == "" {
				missing = append(missing, spec)
			}
		}
	}

	if len(missing) > 0 {
		return MissingArgsError(missing)
	}

	return nil
}

type MissingArgsError []service.ArgSpec

func (e MissingArgsError) Error() string {
	msg := "missing required build arguments:\n"
	for _, spec := range e {
		msg += fmt.Sprintf("  %s:\n", spec.Name)
		msg += fmt.Sprintf("    description: %s\n", spec.Description)
		if spec.Example != "" {
			msg += fmt.Sprintf("    example: %s\n", spec.Example)
		}
	}
	return msg
}
