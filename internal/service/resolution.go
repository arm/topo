package service

type ResolvedArg struct {
	Name  string
	Value string
}

type ResolvedTemplateManifest struct {
	Service map[string]any
	Args    []ResolvedArg
}

func NewResolvedTemplateManifest(sourceManifest TemplateManifest) ResolvedTemplateManifest {
	return ResolvedTemplateManifest{
		Service: sourceManifest.Service,
	}
}
