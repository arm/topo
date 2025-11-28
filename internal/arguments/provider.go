package arguments

type Arg struct {
	Name        string
	Description string
	Required    bool
	Example     string
}

type ResolvedArg struct {
	Name  string
	Value string
}

type Provider interface {
	Provide(args []Arg) ([]ResolvedArg, error)
}
