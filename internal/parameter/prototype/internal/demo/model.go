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
	Example     string
	Required    bool
	Current     *Current
}

type Answer struct {
	Text string
}

func Fixtures() []Parameter {
	return []Parameter{
		{"GREETING", "Greeting to display", "Hello from Topo", true, &Current{"Hello, World", ".env.topo"}},
		{"REGION", "Deployment region", "east", false, &Current{"west", "environment"}},
		{"LABEL", "Project label", "demo", true, nil},
		{"NOTE", "Deployment note", "", false, nil},
		{"EMPTY", "Suffix to append to the greeting", "", true, &Current{"", ".env"}},
	}
}

// Resolve treats a blank draft as no update, including when the current value is empty.
func Resolve(parameter Parameter, answer Answer) (value string, write bool, err error) {
	if answer.Text != "" {
		return answer.Text, true, nil
	}
	if parameter.Required && parameter.Current == nil {
		return "", false, errors.New("value missing: enter a value")
	}
	return "", false, nil
}

func Preview(parameter Parameter, answer Answer) string {
	value, write, err := Resolve(parameter, answer)
	if err != nil {
		return "missing (required)"
	}
	if write {
		return fmt.Sprintf("set %q", value)
	}
	if parameter.Current != nil {
		return fmt.Sprintf("keep %q; no write", parameter.Current.Value)
	}
	return "leave missing; no write"
}

func (p Parameter) Heading(number, total int) string {
	metadata := "optional"
	if p.Required {
		metadata = "required"
	}
	return fmt.Sprintf("%d/%d %s (%s)", number, total, p.Name, metadata)
}

func (p Parameter) Details() string {
	if p.Example == "" {
		return p.Description
	}
	return p.Description + "\nExample: " + p.Example
}

func (p Parameter) Placeholder() string {
	if p.Current == nil {
		return ""
	}
	value := p.Current.Value
	if value == "" {
		value = "<empty string>"
	}
	return fmt.Sprintf("%s  (from %s)", value, p.Current.Source)
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
