package parameter

import (
	"strings"

	"github.com/compose-spec/compose-go/v2/template"
)

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

const assertSuffix = "__TOPO_ASSERT_SATISFIEDBY__"

// AssertSatisfiedBy treats requirements in nested defaults and alternatives as unconditional.
func (p Parameter) AssertSatisfiedBy(val *string) error {
	if val != nil && *val != "" {
		return nil
	}
	for _, reference := range p.References {
		name, expression := p.Name, reference.Expression
		if val != nil {
			// Mark nonempty required (:?) variables with a unique suffix to distinguish them from presence required (?) variables
			suffix := "_" + assertSuffix
			name += suffix
			expression = strings.ReplaceAll(expression, ":?", suffix+":?")
		}
		variables := template.ExtractVariables(map[string]any{"reference": expression}, nil)
		if variables[name].Required {
			return template.MissingRequiredError{Variable: p.Name}
		}
	}
	return nil
}

type Changes map[string]*string

type Resolver interface {
	Resolve(parameters []Parameter) (Changes, error)
}
