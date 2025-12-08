package template

import (
	"github.com/arm-debug/topo-cli/internal/arguments"
)

type ResolvedTemplate struct {
	Service     map[string]any
	ServiceName string
	Args        []arguments.ResolvedArg
}

func Resolve(template Template, argProvider arguments.Provider) (ResolvedTemplate, error) {
	args := make([]arguments.Arg, len(template.Metadata.Args))
	for i, metaArg := range template.Metadata.Args {
		args[i] = arguments.Arg(metaArg)
	}

	resolvedArgs, err := argProvider.Provide(args)
	if err != nil {
		return ResolvedTemplate{}, err
	}

	return ResolvedTemplate{
		Service:     template.Service,
		Args:        resolvedArgs,
		ServiceName: template.ServiceName,
	}, nil
}
