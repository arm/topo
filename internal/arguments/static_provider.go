package arguments

// StaticProvider returns a fixed set of resolved arguments. Useful for testing.
type StaticProvider struct {
	values []ResolvedArg
}

func NewStaticProvider(values ...ResolvedArg) *StaticProvider {
	return &StaticProvider{values: values}
}

func (p *StaticProvider) Provide(args []Arg) ([]ResolvedArg, error) {
	return p.values, nil
}
