package parameter

import "github.com/compose-spec/compose-go/v2/template"

type Parameter struct {
	Name          string
	Description   string
	Example       string
	ExistingValue *string
	References    []Reference
}

type Reference struct {
	Path       string
	Expression string
}

func (p Parameter) needsValue() bool {
	if p.ExistingValue != nil {
		return false
	}
	for _, reference := range p.References {
		variables := template.ExtractVariables(map[string]any{"reference": reference.Expression}, nil)
		if variables[p.Name].Required {
			return true
		}
	}
	return false
}

type Values map[string]string

type Resolver interface {
	Resolve(parameters []Parameter) (Values, error)
}
