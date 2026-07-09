package template

import (
	"github.com/arm/topo/internal/arguments"
)

type ResolvedTemplate struct {
	Services   []Service
	Parameters []arguments.ResolvedArg
}

func Resolve(template Template, argProvider arguments.Provider) (ResolvedTemplate, error) {
	resolvedArgs, err := argProvider.Provide(castParameters(template.Metadata.Parameters))
	if err != nil {
		return ResolvedTemplate{}, err
	}
	return ResolvedTemplate{
		Services:   template.Services,
		Parameters: resolvedArgs,
	}, nil
}

func castParameters(toCast []Parameter) []arguments.Arg {
	casted := make([]arguments.Arg, len(toCast))
	for i, parameter := range toCast {
		casted[i] = arguments.Arg(parameter)
	}
	return casted
}
