package parameter

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/arm/topo/internal/output/term"
)

// InteractiveResolver resolves parameter definitions to values by prompting via stdin/stdout.
type InteractiveResolver struct {
	input  io.Reader
	output io.Writer
}

func NewInteractiveResolver(in io.Reader, out io.Writer) *InteractiveResolver {
	return &InteractiveResolver{input: in, output: out}
}

func (r *InteractiveResolver) Resolve(definitions []Definition, currentValues Values) (Values, error) {
	values := Values{}
	if len(definitions) == 0 {
		return values, nil
	}
	scanner := bufio.NewScanner(r.input)
	palette := term.NewPaletteFor(r.output)

	for i, definition := range definitions {
		prompt := formatParameterPrompt(definition, currentValues, i+1, len(definitions), palette) + "\n"
		if i == 0 {
			prompt = "\n" + prompt
		}
		if _, err := fmt.Fprint(r.output, prompt); err != nil {
			return nil, err
		}

		_, hasCurrentValue := currentValues[definition.Name]

		for {
			if _, err := fmt.Fprint(r.output, palette.Color(term.Cyan, "> ")); err != nil {
				return nil, err
			}
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return nil, err
				}
				return values, nil
			}
			value := strings.TrimSpace(scanner.Text())
			if value != "" {
				values[definition.Name] = value
				break
			}
			if !definition.Required || hasCurrentValue {
				break
			}
			if _, err := fmt.Fprintln(r.output, palette.Color(term.Red, "! A value is required.")); err != nil {
				return nil, err
			}
		}
	}
	return values, nil
}

func formatParameterPrompt(definition Definition, currentValues Values, number, total int, palette term.Palette) string {
	currentValue, hasCurrentValue := currentValues[definition.Name]
	question := "Set " + definition.Name + "?"
	instruction := "Enter a value, or leave blank to skip."
	if definition.Required {
		instruction = "Enter a value."
	}
	if hasCurrentValue {
		question = "Change " + definition.Name + "?"
		instruction = "Enter a new value, or leave blank to keep the current value."
	}

	progress := palette.Color(term.Dim, fmt.Sprintf("(%d/%d)", number, total))
	lines := []string{palette.Color(term.Bold, question) + " " + progress}
	if description := strings.TrimSpace(definition.Description); description != "" {
		lines = append(lines, description)
	}
	var metadata []string
	if hasCurrentValue {
		metadata = append(metadata, fmt.Sprintf("%s %q", palette.Color(term.Gray, "Current:"), currentValue))
	}
	if example := strings.TrimSpace(definition.Example); example != "" {
		metadata = append(metadata, fmt.Sprintf("%s %q", palette.Color(term.Gray, "Example:"), example))
	}
	if len(metadata) > 0 {
		lines = append(lines, "")
		lines = append(lines, metadata...)
	}
	lines = append(lines, "", palette.Color(term.Cyan, instruction))
	return strings.Join(lines, "\n")
}
