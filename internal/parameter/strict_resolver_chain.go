package parameter

import (
	"fmt"
	"maps"
	"strings"
)

// StrictResolverChain chains resolvers and validates parameter values against their references.
// Each resolver receives only parameters not supplied by earlier resolvers.
type StrictResolverChain struct {
	resolvers []Resolver
}

func NewStrictResolverChain(resolvers ...Resolver) *StrictResolverChain {
	return &StrictResolverChain{resolvers: resolvers}
}

func (r *StrictResolverChain) Resolve(parameters []Parameter) (Changes, error) {
	changes := Changes{}
	remaining := parameters

	for _, resolver := range r.resolvers {
		if len(remaining) == 0 {
			break
		}

		resolvedChanges, err := resolver.Resolve(remaining)
		if err != nil {
			return nil, err
		}

		maps.Copy(changes, resolvedChanges)
		remaining = withoutChanges(remaining, changes)
	}

	if err := validateRequiredValues(parameters, changes); err != nil {
		return nil, err
	}

	return changes, nil
}

type validationError struct {
	Parameter Parameter
	Err       error
}

type ValidationErrors []validationError

func (e ValidationErrors) Error() string {
	var msg strings.Builder
	msg.WriteString("parameter validation failed:\n")
	for _, ve := range e {
		parameter := ve.Parameter
		fmt.Fprintf(&msg, "%s:\n", parameter.Name)
		if parameter.Description != "" {
			fmt.Fprintf(&msg, "  description: %s\n", parameter.Description)
		}
		if parameter.Example != "" {
			fmt.Fprintf(&msg, "  example: %s\n", parameter.Example)
		}
		fmt.Fprintf(&msg, "  references:\n")
		for _, ref := range parameter.References {
			fmt.Fprintf(&msg, "    - %s: %s\n", ref.Path, ref.Expression)
		}
		fmt.Fprintf(&msg, "  reason: %v\n", ve.Err)
	}
	return strings.TrimSpace(msg.String())
}

func withoutChanges(parameters []Parameter, changes Changes) []Parameter {
	var remaining []Parameter
	for _, parameter := range parameters {
		if _, exists := changes[parameter.Name]; !exists {
			remaining = append(remaining, parameter)
		}
	}
	return remaining
}

func validateRequiredValues(parameters []Parameter, changes Changes) error {
	var errs []validationError
	for _, parameter := range parameters {
		value := parameter.ExistingValue
		newValue, supplied := changes[parameter.Name]
		if supplied {
			value = newValue
		}

		if err := parameter.AssertSatisfiedBy(value); err != nil {
			errs = append(errs, validationError{Parameter: parameter, Err: err})
		}
	}

	if len(errs) > 0 {
		return ValidationErrors(errs)
	}

	return nil
}
