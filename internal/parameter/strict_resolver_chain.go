package parameter

import (
	"fmt"
	"maps"
	"slices"
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

func (r *StrictResolverChain) Resolve(definitions []Definition) (Values, error) {
	values := Values{}
	remaining := definitions

	for _, resolver := range r.resolvers {
		if len(remaining) == 0 {
			break
		}

		newValues, err := resolver.Resolve(remaining)
		if err != nil {
			return nil, err
		}

		maps.Copy(values, newValues)
		remaining = withoutValues(remaining, values)

		if allRequiredHaveValues(definitions, values) {
			break
		}
	}

	if err := validateRequiredValues(definitions, values); err != nil {
		return nil, err
	}

	return values, nil
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

func withoutValues(definitions []Definition, values Values) []Definition {
	var remaining []Definition
	for _, definition := range definitions {
		if _, exists := values[definition.Name]; !exists {
			remaining = append(remaining, definition)
		}
	}
	return remaining
}

func allRequiredHaveValues(definitions []Definition, values Values) bool {
	for _, definition := range definitions {
		if definition.Required && !hasValue(definition, values) {
			return false
		}
	}
	return true
}

func hasValue(definition Definition, values Values) bool {
	if value, exists := values[definition.Name]; exists {
		return value != ""
	}
	if len(definition.CurrentValues) == 0 {
		return false
	}
	return !slices.Contains(definition.CurrentValues, "")
}

func validateRequiredValues(definitions []Definition, values Values) error {
	var missing []Definition
	for _, definition := range definitions {
		if definition.Required && !hasValue(definition, values) {
			missing = append(missing, definition)
		}
	}

	if len(missing) > 0 {
		return MissingParametersError(missing)
	}

	return nil
}
