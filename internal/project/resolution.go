package project

import (
	"github.com/arm/topo/internal/arguments"
)

func Resolve(p Project, argProvider arguments.Provider) ([]arguments.ResolvedArg, error) {
	resolvedArgs, err := argProvider.Provide(castParameters(p.Metadata.Parameters))
	if err != nil {
		return nil, err
	}
	return resolvedArgs, nil
}

func castParameters(toCast []Parameter) []arguments.Arg {
	casted := make([]arguments.Arg, len(toCast))
	for i, parameter := range toCast {
		casted[i] = arguments.Arg(parameter)
	}
	return casted
}
