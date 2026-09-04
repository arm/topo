package parameter

// StaticProvider returns a fixed set of parameter values. Useful for testing.
type StaticProvider struct {
	values Values
}

func NewStaticProvider(values Values) *StaticProvider {
	return &StaticProvider{values: values}
}

func (p *StaticProvider) Provide(_ []Definition) (Values, error) {
	return p.values, nil
}
