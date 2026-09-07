package parameter

// StaticResolver returns a fixed set of parameter values. Useful for testing.
type StaticResolver struct {
	values Values
}

func NewStaticResolver(values Values) *StaticResolver {
	return &StaticResolver{values: values}
}

func (r *StaticResolver) Resolve(_ []Definition) (Values, error) {
	return r.values, nil
}
