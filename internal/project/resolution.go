package project

import (
	"github.com/arm/topo/internal/arguments"
)

func Resolve(p Project, argProvider arguments.Provider) ([]arguments.ResolvedArg, error) {
	resolvedArgs, err := argProvider.Provide(castParameters(p.Metadata.Parameters, p.currentParameterValues))
	if err != nil {
		return nil, err
	}
	return resolvedArgs, nil
}

func castParameters(parameters []Parameter, currentValues map[string][]string) []arguments.Arg {
	casted := make([]arguments.Arg, len(parameters))
	for i, parameter := range parameters {
		casted[i] = arguments.Arg{
			Name:          parameter.Name,
			Description:   parameter.Description,
			Required:      parameter.Required,
			Example:       parameter.Example,
			Default:       parameter.Default,
			CurrentValues: currentValues[parameter.Name],
		}
	}
	return casted
}
