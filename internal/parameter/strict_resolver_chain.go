package parameter

import (
	"fmt"
	"maps"
	"strings"
)

// StrictResolverChain chains resolvers and ensures all required parameters have values.
// Each resolver receives only parameters not supplied by earlier resolvers.
type StrictResolverChain struct {
	resolvers []Resolver
}

func NewStrictResolverChain(resolvers ...Resolver) *StrictResolverChain {
	return &StrictResolverChain{resolvers: resolvers}
}

func (r *StrictResolverChain) Resolve(parameters []Parameter) (Values, error) {
	updates := Values{}
	remaining := parameters

	for _, resolver := range r.resolvers {
		if len(remaining) == 0 {
			break
		}

		newValues, err := resolver.Resolve(remaining)
		if err != nil {
			return nil, err
		}

		maps.Copy(updates, newValues)
		remaining = withoutValues(remaining, updates)
	}

	if err := validateRequiredValues(parameters, updates); err != nil {
		return nil, err
	}

	return updates, nil
}

type MissingParametersError []Parameter

func (e MissingParametersError) Error() string {
	var msg strings.Builder
	msg.WriteString("missing value(s) for required parameters:\n")
	for _, parameter := range e {
		fmt.Fprintf(&msg, "  %s:\n", parameter.Name)
		fmt.Fprintf(&msg, "    description: %s\n", parameter.Description)
		if parameter.Example != "" {
			fmt.Fprintf(&msg, "    example: %s\n", parameter.Example)
		}
	}
	return msg.String()
}

func withoutValues(parameters []Parameter, values Values) []Parameter {
	var remaining []Parameter
	for _, parameter := range parameters {
		if _, exists := values[parameter.Name]; !exists {
			remaining = append(remaining, parameter)
		}
	}
	return remaining
}

func validateRequiredValues(parameters []Parameter, updates Values) error {
	var missing []Parameter
	for _, parameter := range parameters {
		_, supplied := updates[parameter.Name]
		if !supplied && parameter.Required && parameter.ExistingValue == nil {
			missing = append(missing, parameter)
		}
	}

	if len(missing) > 0 {
		return MissingParametersError(missing)
	}

	return nil
}
