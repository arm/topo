package project

import (
	"fmt"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/parameter"
)

func Configure(scope Scope, resolver parameter.Resolver) (parameter.Changes, error) {
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

	changes, err := resolver.Resolve(parameters)
	if err != nil {
		return nil, err
	}

	if len(changes) == 0 {
		return nil, nil
	}

	return changes, nil
}
