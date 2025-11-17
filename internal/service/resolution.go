package service

import (
	"github.com/arm-debug/topo-cli/internal/arguments"
)

type ResolvedTemplateManifest struct {
	Service map[string]any
	Args    []arguments.ResolvedArg
}

func ResolveTemplateManifest(sourceManifest TemplateManifest, argCollector arguments.Collector) (ResolvedTemplateManifest, error) {
	args := make([]arguments.Arg, len(sourceManifest.Metadata.Args))
	for i, metaArg := range sourceManifest.Metadata.Args {
		args[i] = arguments.Arg(metaArg)
	}

	resolvedArgs, err := argCollector.Collect(args)
	if err != nil {
		return ResolvedTemplateManifest{}, err
	}

	return ResolvedTemplateManifest{
		Service: sourceManifest.Service,
		Args:    resolvedArgs,
	}, nil
}
