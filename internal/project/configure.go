package project

import (
	"fmt"
	"maps"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
)

func resolveFilePathParameters(scope Scope, resolver parameter.Resolver, resolved map[string]bool) (parameter.Changes, error) {
	sources, err := readRootSources(scope)
	if err != nil {
		return nil, err
	}
	changes := parameter.Changes{}
	for index := 0; index < len(sources); index++ {
		declared, err := loadParameterDeclarations(sources)
		if err != nil {
			return nil, err
		}
		source := &sources[index]
		selected := selectFilePathParameters(declared, []composeSource{*source}, source.environment)
		updates, err := resolveSelectedFilePathParameters(declared, selected, resolver, resolved)
		if err != nil {
			return nil, err
		}
		maps.Copy(changes, updates)
		for name, value := range changes {
			if value != nil {
				source.environment[name] = *value
			} else {
				delete(source.environment, name)
			}
		}
		children, err := readDirectIncludes(*source)
		if err != nil {
			return nil, err
		}
		sources = append(sources, children...)
		if len(declared) == 0 {
			continue
		}
		extended, err := readDirectExtendedServices(*source, sources)
		if err != nil {
			return nil, err
		}
		sources = append(sources, extended...)
	}
	return changes, nil
}

func resolveSelectedFilePathParameters(declared, selected []parameter.Parameter, resolver parameter.Resolver, resolved map[string]bool) (parameter.Changes, error) {
	var pending []parameter.Parameter
	for _, selectedParameter := range selected {
		if !resolved[selectedParameter.Name] {
			pending = append(pending, selectedParameter)
		}
	}
	selected = pending
	if len(selected) == 0 {
		return nil, nil
	}
	var changes parameter.Changes
	var err error
	if staged, ok := resolver.(parameter.SelectedParameterResolver); ok {
		changes, err = staged.ResolveSelected(declared, selected)
	} else {
		changes, err = resolver.Resolve(selected)
	}
	if err != nil {
		return nil, err
	}
	for _, selectedParameter := range selected {
		resolved[selectedParameter.Name] = true
	}
	return changes, nil
}

func Configure(scope Scope, resolver parameter.Resolver) (parameter.Changes, error) {
	resolved := map[string]bool{}
	pathChanges, err := resolveFilePathParameters(scope, resolver, resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path parameters: %w", err)
	}
	scope.EnvOverrides = pathChanges
	parameters, err := LoadParameters(scope)
	if err != nil {
		return nil, fmt.Errorf("failed to load parameters: %w", err)
	}

	for i := range parameters {
		param := &parameters[i]
		if len(param.References) == 0 {
			logger.Warn(fmt.Sprintf("parameter %q is not referenced through an environment variable in the Compose file; configuring it will have no effect", param.Name))
		}
	}

	var remaining []parameter.Parameter
	for _, declared := range parameters {
		if !resolved[declared.Name] {
			remaining = append(remaining, declared)
		}
	}
	var changes parameter.Changes
	if staged, ok := resolver.(parameter.SelectedParameterResolver); ok {
		changes, err = staged.ResolveSelected(parameters, remaining)
	} else {
		changes, err = resolver.Resolve(remaining)
	}
	if err != nil {
		return nil, err
	}
	combined := parameter.Changes{}
	maps.Copy(combined, pathChanges)
	maps.Copy(combined, changes)
	changes, err = parameter.NewStrictResolverChain(parameter.NewStaticResolver(combined)).Resolve(parameters)
	if err != nil {
		return nil, err
	}

	if len(changes) == 0 {
		return nil, nil
	}

	return changes, nil
}
