package parameter

// StaticProvider returns a fixed set of parameter values. Useful for testing.
type StaticProvider struct {
	values []Provided
}

func NewStaticProvider(values ...Provided) *StaticProvider {
	return &StaticProvider{values: values}
}

func (p *StaticProvider) Provide(_ []Definition) ([]Provided, error) {
	return p.values, nil
}
