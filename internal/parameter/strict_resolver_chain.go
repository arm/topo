package parameter

import (
	"fmt"
	"maps"
	"strings"
)

// StrictResolverChain chains resolvers and ensures all required parameters have values.
// It stops early once all required parameters are satisfied.
type StrictResolverChain struct {
	resolvers []Resolver
}

func NewStrictResolverChain(resolvers ...Resolver) *StrictResolverChain {
	return &StrictResolverChain{resolvers: resolvers}
}

func (r *StrictResolverChain) Resolve(definitions []Definition, currentValues Values) (Values, error) {
	updates := Values{}
	remaining := definitions

	for _, resolver := range r.resolvers {
		if len(remaining) == 0 {
			break
		}

		newValues, err := resolver.Resolve(remaining, currentValues)
		if err != nil {
			return nil, err
		}

		maps.Copy(updates, newValues)
		remaining = withoutValues(remaining, updates)

		if allRequiredHaveValues(definitions, updates, currentValues) {
			break
		}
	}

	if err := validateRequiredValues(definitions, updates, currentValues); err != nil {
		return nil, err
	}

	return updates, nil
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
	}
	return msg.String()
}

func withoutValues(definitions []Definition, values Values) []Definition {
	var remaining []Definition
	for _, definition := range definitions {
		if _, exists := values[definition.Name]; !exists {
			remaining = append(remaining, definition)
		}
	}
	return remaining
}

func allRequiredHaveValues(definitions []Definition, updates, currentValues Values) bool {
	for _, definition := range definitions {
		if definition.Required && !hasValue(definition.Name, updates, currentValues) {
			return false
		}
	}
	return true
}

func hasValue(name string, updates, currentValues Values) bool {
	value, supplied := updates[name]
	if !supplied {
		value = currentValues[name]
	}
	return value != ""
}

func validateRequiredValues(definitions []Definition, updates, currentValues Values) error {
	var missing []Definition
	for _, definition := range definitions {
		if definition.Required && !hasValue(definition.Name, updates, currentValues) {
			missing = append(missing, definition)
		}
	}

	if len(missing) > 0 {
		return MissingParametersError(missing)
	}

	return nil
}
