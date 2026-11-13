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
	return r.ResolveSelected(parameters, parameters)
}

func (r *StrictResolverChain) ResolveSelected(declared, selected []Parameter) (Changes, error) {
	changes := Changes{}
	remaining := selected

	for _, resolver := range r.resolvers {
		if len(remaining) == 0 {
			break
		}

		var resolvedChanges Changes
		var err error
		if staged, ok := resolver.(SelectedParameterResolver); ok {
			resolvedChanges, err = staged.ResolveSelected(declared, remaining)
		} else {
			resolvedChanges, err = resolver.Resolve(remaining)
		}
		if err != nil {
			return nil, err
		}

		maps.Copy(changes, resolvedChanges)
		remaining = withoutChanges(remaining, changes)
	}

	if err := validateRequiredValues(selected, changes); err != nil {
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
	values := Changes{}
	for _, parameter := range parameters {
		if parameter.ExistingValue != nil {
			values[parameter.Name] = parameter.ExistingValue
		}
	}
	maps.Copy(values, changes)
	for _, parameter := range parameters {
		for _, reference := range parameter.References {
			if err := reference.interpolate(values); err != nil {
				return fmt.Errorf("parameter validation failed at %q: %w", reference.Path, err)
			}
		}
	}

	return nil
}
