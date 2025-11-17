package arguments

import (
	"fmt"
	"maps"
)

type Arg struct {
	Name        string
	Description string
	Required    bool
	Example     string
}

type ResolvedArg struct {
	Name  string
	Value string
}

type Provider interface {
	Provide(args []Arg) (map[string]string, error)
	Name() string
}

type Collector []Provider

func NewCollector(providers ...Provider) Collector {
	return Collector(providers)
}

func (c Collector) Collect(args []Arg) ([]ResolvedArg, error) {
	provided := make(map[string]string)
	remaining := args

	for _, provider := range c {
		if len(remaining) == 0 {
			break
		}

		values, err := provider.Provide(remaining)
		if err != nil {
			return nil, fmt.Errorf("%s provider failed: %w", provider.Name(), err)
		}
		maps.Copy(provided, values)

		if allRequiredProvided(args, provided) {
			break
		}

		remaining = filterProvided(remaining, values)
	}

	if err := validateRequiredProvided(args, provided); err != nil {
		return nil, err
	}

	resolved := []ResolvedArg{}
	for _, arg := range args {
		if value, ok := provided[arg.Name]; ok {
			resolved = append(resolved, ResolvedArg{
				Name:  arg.Name,
				Value: value,
			})
		}
	}

	return resolved, nil
}

func filterProvided(args []Arg, provided map[string]string) []Arg {
	var remaining []Arg
	for _, arg := range args {
		if _, exists := provided[arg.Name]; !exists {
			remaining = append(remaining, arg)
		}
	}
	return remaining
}

func allRequiredProvided(args []Arg, provided map[string]string) bool {
	for _, arg := range args {
		if arg.Required {
			if value, exists := provided[arg.Name]; !exists || value == "" {
				return false
			}
		}
	}
	return true
}

func validateRequiredProvided(args []Arg, provided map[string]string) error {
	var missing []Arg
	for _, arg := range args {
		if arg.Required {
			if value, exists := provided[arg.Name]; !exists || value == "" {
				missing = append(missing, arg)
			}
		}
	}

	if len(missing) > 0 {
		return MissingArgsError(missing)
	}

	return nil
}

type MissingArgsError []Arg

func (e MissingArgsError) Error() string {
	msg := "missing required build arguments:\n"
	for _, arg := range e {
		msg += fmt.Sprintf("  %s:\n", arg.Name)
		msg += fmt.Sprintf("    description: %s\n", arg.Description)
		if arg.Example != "" {
			msg += fmt.Sprintf("    example: %s\n", arg.Example)
		}
	}
	return msg
}
