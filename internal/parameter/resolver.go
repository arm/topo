package parameter

type Definition struct {
	Name        string
	Description string
	Required    bool
	Example     string
}

type Values map[string]string

type Resolver interface {
	Resolve(definitions []Definition, currentValues Values) (Values, error)
}
