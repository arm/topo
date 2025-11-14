package service

type ResolvedArg struct {
	Name  string
	Value string
}

type ResolvedTemplateManifest struct {
	Service map[string]any
	Args    []ResolvedArg
}

type ArgumentCollector interface {
	Collect(specs []ArgSpec) (map[string]string, error)
}

func NewResolvedTemplateManifest(sourceManifest TemplateManifest, argCollector ArgumentCollector) (ResolvedTemplateManifest, error) {
	buildArgs, err := argCollector.Collect(sourceManifest.Metadata.Args)
	if err != nil {
		return ResolvedTemplateManifest{}, err
	}

	args := make([]ResolvedArg, 0, len(buildArgs))
	for _, spec := range sourceManifest.Metadata.Args {
		if value, ok := buildArgs[spec.Name]; ok {
			args = append(args, ResolvedArg{
				Name:  spec.Name,
				Value: value,
			})
		}
	}

	return ResolvedTemplateManifest{
		Service: sourceManifest.Service,
		Args:    args,
	}, nil
}
