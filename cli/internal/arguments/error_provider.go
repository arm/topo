package arguments

// ErrorProvider always returns an error. Useful for testing error handling.
type ErrorProvider struct {
	err error
}

func NewErrorProvider(err error) *ErrorProvider {
	return &ErrorProvider{err: err}
}

func (p *ErrorProvider) Provide(args []Arg) ([]ResolvedArg, error) {
	return nil, p.err
}
