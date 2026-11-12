package parameter

import (
	"errors"
	"maps"

	"github.com/compose-spec/compose-go/v2/template"
)

type Source struct {
	Provider string
	Group    string
	Fact     string
}

type Parameter struct {
	Name          string
	Source        *Source
	Description   string
	Example       string
	ExistingValue *string
	References    []Reference
}

type Reference struct {
	Path        string
	Expression  string
	Environment map[string]string
}

func (p Parameter) AssertSatisfiedBy(val *string, changes Changes) error {
	if val != nil && *val != "" {
		return nil
	}
	for _, reference := range p.References {
		referenceValue := val
		if _, supplied := changes[p.Name]; referenceValue == nil && !supplied {
			if value, present := reference.Environment[p.Name]; present {
				referenceValue = &value
			}
		}
		if referenceValue != nil && *referenceValue != "" {
			continue
		}
		values := Changes{p.Name: referenceValue}
		maps.Copy(values, changes)
		err := reference.interpolate(values)
		if err == nil {
			continue
		}
		var missing *template.MissingRequiredError
		if errors.As(err, &missing) && missing.Variable != p.Name {
			continue
		}
		return err
	}
	return nil
}

type Changes map[string]*string

func (reference Reference) interpolate(changes Changes) error {
	return reference.interpolateWithLookup(changes, nil)
}

func (reference Reference) interpolateWithLookup(changes Changes, onLookup func(string)) error {
	_, err := template.SubstituteWithOptions(reference.Expression, func(name string) (string, bool) {
		if onLookup != nil {
			onLookup(name)
		}
		if value, supplied := changes[name]; supplied {
			if value == nil {
				return "", false
			}
			return *value, true
		}
		value, present := reference.Environment[name]
		return value, present
	}, template.WithoutLogging)
	return err
}

func (parameter Parameter) isUsed(changes Changes) bool {
	if len(parameter.References) == 0 {
		return true
	}
	for _, reference := range parameter.References {
		used := false
		err := reference.interpolateWithLookup(changes, func(name string) {
			used = used || name == parameter.Name
		})
		if used || err != nil {
			return true
		}
	}
	return false
}

type Resolver interface {
	Resolve(parameters []Parameter) (Changes, error)
}
