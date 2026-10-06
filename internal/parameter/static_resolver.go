package parameter

// StaticResolver returns a fixed set of parameter changes. Useful for testing.
type StaticResolver struct {
	changes Changes
}

func NewStaticResolver(changes Changes) *StaticResolver {
	return &StaticResolver{changes: changes}
}

func (r *StaticResolver) Resolve(_ []Parameter) (Changes, error) {
	return r.changes, nil
}
