package project

import (
	"github.com/arm/topo/cli/internal/arguments"
)

type ResolvedProject struct {
	Services   []Service
	Parameters []arguments.ResolvedArg
}

func Resolve(p Project, argProvider arguments.Provider) (ResolvedProject, error) {
	resolvedArgs, err := argProvider.Provide(castParameters(p.Metadata.Parameters))
	if err != nil {
		return ResolvedProject{}, err
	}
	return ResolvedProject{
		Services:   p.Services,
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
