package parameter

import (
	"fmt"
	"slices"
	"strings"
)

// StrictProviderChain chains providers and ensures all required parameters are resolved.
// It stops early once all required parameters are satisfied.
type StrictProviderChain struct {
	providers []Provider
}

func NewStrictProviderChain(providers ...Provider) *StrictProviderChain {
	return &StrictProviderChain{providers: providers}
}

func (p *StrictProviderChain) Provide(definitions []Definition) ([]Provided, error) {
	values := make(map[string]string)
	remaining := definitions

	for _, provider := range p.providers {
		if len(remaining) == 0 {
			break
		}

		provided, err := provider.Provide(remaining)
		if err != nil {
			return nil, err
		}

		for _, entry := range provided {
			values[entry.Name] = entry.Value
		}

		remaining = filterProvided(remaining, values)

		if allRequiredResolved(definitions, values) {
			break
		}
	}

	if err := validateRequiredResolved(definitions, values); err != nil {
		return nil, err
	}

	var provided []Provided
	for _, definition := range definitions {
		if value, ok := values[definition.Name]; ok {
			provided = append(provided, Provided{Name: definition.Name, Value: value})
		}
	}

	return provided, nil
}

type MissingParametersError []Definition

func (e MissingParametersError) Error() string {
	var msg strings.Builder
	msg.WriteString("missing value(s) for required parameters:\n")
	for _, definition := range e {
		fmt.Fprintf(&msg, "  %s:\n", definition.Name)
		fmt.Fprintf(&msg, "    description: %s\n", definition.Description)
		if definition.Example != "" {
			fmt.Fprintf(&msg, "    example: %s\n", definition.Example)
		}
		if len(definition.CurrentValues) > 0 {
			fmt.Fprintf(&msg, "    # current: %s\n", formatCurrentValues(definition.CurrentValues))
		}
	}
	return msg.String()
}

func filterProvided(definitions []Definition, values map[string]string) []Definition {
	var remaining []Definition
	for _, definition := range definitions {
		if _, exists := values[definition.Name]; !exists {
			remaining = append(remaining, definition)
		}
	}
	return remaining
}

func allRequiredResolved(definitions []Definition, values map[string]string) bool {
	for _, definition := range definitions {
		if definition.Required && !isResolved(definition, values) {
			return false
		}
	}
	return true
}

func isResolved(definition Definition, values map[string]string) bool {
	if value, exists := values[definition.Name]; exists {
		return value != ""
	}
	if len(definition.CurrentValues) == 0 {
		return false
	}
	return !slices.Contains(definition.CurrentValues, "")
}

func validateRequiredResolved(definitions []Definition, values map[string]string) error {
	var missing []Definition
	for _, definition := range definitions {
		if definition.Required && !isResolved(definition, values) {
			missing = append(missing, definition)
		}
	}

	if len(missing) > 0 {
		return MissingParametersError(missing)
	}

	return nil
}
