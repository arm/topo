package project

import (
	"fmt"
	"slices"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
	"github.com/compose-spec/compose-go/v2/template"
)

func Configure(scope Scope, resolver parameter.Resolver) (map[string]string, error) {
	parameters, err := LoadParameters(scope)
	if err != nil {
		return nil, fmt.Errorf("failed to load parameters: %w", err)
	}

	if len(parameters) == 0 {
		return nil, nil
	}

	if err := warnUnreferencedParameters(scope.ComposeFile, parameters); err != nil {
		return nil, err
	}

	values, err := resolver.Resolve(parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to collect parameter values: %w", err)
	}

	if len(values) == 0 {
		return nil, nil
	}

	return values, nil
}

func warnUnreferencedParameters(composeFilePath string, definitions []parameter.Parameter) error {
	model, err := readUninterpolated(composeFilePath)
	if err != nil {
		return err
	}
	referencedEnvVars := template.ExtractVariables(model, nil)

	unreferencedDefinitions := slices.DeleteFunc(slices.Clone(definitions), func(d parameter.Parameter) bool {
		_, referenced := referencedEnvVars[d.Name]
		return referenced
	})

	for _, param := range unreferencedDefinitions {
		logger.Warn(fmt.Sprintf("parameter %q is not referenced through an environment variable in the Compose file; configuring it will have no effect", param.Name))
	}

	return nil
}
