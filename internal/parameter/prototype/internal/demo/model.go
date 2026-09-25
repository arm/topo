// Package demo contains the in-memory fixtures and semantics shared by the throwaway prompts.
package demo

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"
)

type Current struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

type Parameter struct {
	Name        string
	Description string
	Required    bool
	Current     *Current
}

type Answer struct {
	Text  string
	Empty bool
}

func Fixtures() []Parameter {
	return []Parameter{
		{"GREETING", "Greeting to display", true, &Current{"Hello, World", ".env.topo"}},
		{"REGION", "Deployment region", false, &Current{"west", "environment"}},
		{"LABEL", "Required but currently missing. Try Enter, then Ctrl+X.", true, nil},
		{"NOTE", "Optional note. Enter leaves this missing.", false, nil},
		{"EMPTY", "Required and already present as an empty string.", true, &Current{"", ".env"}},
	}
}

// Resolve returns whether an update is needed, independently of the value's length.
func Resolve(parameter Parameter, answer Answer) (value string, write bool, err error) {
	if answer.Empty {
		return "", true, nil
	}
	if answer.Text != "" {
		return answer.Text, true, nil
	}
	if parameter.Required && parameter.Current == nil {
		return "", false, errors.New("value missing: type a value or use Ctrl+X to set empty")
	}
	return "", false, nil
}

func Preview(parameter Parameter, answer Answer) string {
	value, write, err := Resolve(parameter, answer)
	if err != nil {
		return "missing (required)"
	}
	if write {
		if value == "" {
			return "set empty"
		}
		return fmt.Sprintf("set %q", value)
	}
	if parameter.Current != nil {
		return fmt.Sprintf("keep %q; no write", parameter.Current.Value)
	}
	return "leave missing; no write"
}

func (p Parameter) Heading() string {
	requirement := "optional"
	if p.Required {
		requirement = "required"
	}
	source := "missing"
	if p.Current != nil {
		source = "current from " + p.Current.Source
	}
	return fmt.Sprintf("%s (%s; %s)", p.Name, requirement, source)
}

func (p Parameter) Placeholder(answer Answer) string {
	if answer.Empty {
		return "<set empty>"
	}
	if p.Current == nil {
		return ""
	}
	if p.Current.Value == "" {
		return "<empty current value>"
	}
	return p.Current.Value
}

func ValidateText(text string) error {
	if !utf8.ValidString(text) {
		return errors.New("invalid UTF-8 input rejected")
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return errors.New("single-line input only: paste containing control characters was rejected")
		}
	}
	return nil
}
