package template

import (
	"github.com/arm/topo/internal/arguments"
)

type ResolvedTemplate struct {
	Services []Service
	Args     []arguments.ResolvedArg
}

func Resolve(template Template, argProvider arguments.Provider) (ResolvedTemplate, error) {
	resolvedArgs, err := argProvider.Provide(castArgs(template.Metadata.Args))
	if err != nil {
		return ResolvedTemplate{}, err
	}
	return ResolvedTemplate{
		Services: template.Services,
		Args:     resolvedArgs,
	}, nil
}

func castArgs(toCast []Arg) []arguments.Arg {
	casted := make([]arguments.Arg, len(toCast))
	for i, metaArg := range toCast {
		casted[i] = arguments.Arg(metaArg)
	}
	return casted
}
