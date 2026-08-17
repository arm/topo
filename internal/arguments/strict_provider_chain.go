package arguments

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

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

		if allRequiredResolved(args, provided) {
			break
		}
	}

	if err := validateRequiredResolved(args, provided); err != nil {
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
	msg.WriteString("missing value(s) for required parameters:\n")
	for _, arg := range e {
		fmt.Fprintf(&msg, "  %s:\n", arg.Name)
		fmt.Fprintf(&msg, "    description: %s\n", arg.Description)
		if arg.Example != "" {
			fmt.Fprintf(&msg, "    example: %s\n", arg.Example)
		}
		currentValues, err := json.Marshal(arg.CurrentValues)
		if err == nil {
			fmt.Fprintf(&msg, "    # current: %s\n", currentValues)
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

func allRequiredResolved(args []Arg, provided map[string]string) bool {
	for _, arg := range args {
		if arg.Required && !isResolved(arg, provided) {
			return false
		}
	}
	return true
}

func isResolved(arg Arg, provided map[string]string) bool {
	if value, exists := provided[arg.Name]; exists {
		return value != ""
	}
	if len(arg.CurrentValues) == 0 {
		return false
	}
	return !slices.Contains(arg.CurrentValues, "")
}

func validateRequiredResolved(args []Arg, provided map[string]string) error {
	var missing []Arg
	for _, arg := range args {
		if arg.Required && !isResolved(arg, provided) {
			missing = append(missing, arg)
		}
	}

	if len(missing) > 0 {
		return MissingArgsError(missing)
	}

	return nil
}
