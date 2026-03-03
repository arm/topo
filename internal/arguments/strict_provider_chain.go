package arguments

import "strings"

import "fmt"

// StrictProviderChain chains multiple providers and ensures all required arguments are resolved.
// It stops early once all required arguments are satisfied.
type StrictProviderChain struct {
	providers []Provider
}

func NewStrictProviderChain(providers ...Provider) *StrictProviderChain {
	return &StrictProviderChain{providers: providers}
}

func (p *StrictProviderChain) Provide(args []Arg) ([]ResolvedArg, error) {
	provided := make(map[string]string)
	remaining := args

	for _, provider := range p.providers {
		if len(remaining) == 0 {
			break
		}

		resolved, err := provider.Provide(remaining)
		if err != nil {
			return nil, err
		}

		for _, r := range resolved {
			provided[r.Name] = r.Value
		}

		remaining = filterProvided(remaining, provided)

		if allRequiredProvided(args, provided) {
			break
		}
	}

	if len(remaining) > 0 {
		defaultNonProvided(remaining, provided)
	}

	if err := validateRequiredProvided(args, provided); err != nil {
		return nil, err
	}

	var result []ResolvedArg
	for _, arg := range args {
		if value, ok := provided[arg.Name]; ok {
			result = append(result, ResolvedArg{Name: arg.Name, Value: value})
		}
	}

	return result, nil
}

type MissingArgsError []Arg

func (e MissingArgsError) Error() string {
	var msg strings.Builder
	msg.WriteString("missing required build arguments:\n")
	for _, arg := range e {
		fmt.Fprintf(&msg, "  %s:\n", arg.Name)
		fmt.Fprintf(&msg, "    description: %s\n", arg.Description)
		if arg.Example != "" {
			fmt.Fprintf(&msg, "    example: %s\n", arg.Example)
		}
	}
	return msg.String()
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

func defaultNonProvided(remaining []Arg, provided map[string]string) {
	for _, arg := range remaining {
		if arg.Default != "" {
			provided[arg.Name] = arg.Default
		}
	}
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
