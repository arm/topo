package parameter

type Parameter struct {
	Name          string
	Description   string
	Required      bool
	Example       string
	ExistingValue *string
}

type Values map[string]string

type Resolver interface {
	Resolve(parameters []Parameter) (Values, error)
}
