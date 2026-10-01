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
		prompt := fmt.Sprintf("%s\n", formatParameterPrompt(definition, currentValues, i+1, len(definitions), palette))
		if _, err := fmt.Fprint(r.output, prompt); err != nil {
			return nil, err
		}

		_, hasCurrentValue := currentValues[definition.Name]

		for {
			if _, err := fmt.Fprintf(r.output, "%s ", palette.Color(term.Magenta, ">")); err != nil {
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
			if _, err := fmt.Fprintf(r.output, "%s A value is required.\n", palette.Color(term.Red, "✗")); err != nil {
				return nil, err
			}
		}
		if _, err := fmt.Fprintln(r.output); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func formatParameterPrompt(definition Definition, currentValues Values, number, total int, palette term.Palette) string {
	currentValue, hasCurrentValue := currentValues[definition.Name]
	progress := palette.Color(term.Dim, fmt.Sprintf("%d/%d", number, total))
	lines := []string{fmt.Sprintf("%s %s", progress, definition.Name), ""}
	if description := strings.TrimSpace(definition.Description); description != "" {
		lines = append(lines, fmt.Sprintf("    %s", strings.ReplaceAll(description, "\n", "\n    ")), "")
	}
	var metadata []string
	if hasCurrentValue {
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Current:"), currentValue))
	}
	if example := strings.TrimSpace(definition.Example); example != "" {
		indentedExample := strings.ReplaceAll(example, "\n", "\n    ")
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Example:"), indentedExample))
	}
	if len(metadata) > 0 {
		lines = append(lines, metadata...)
		lines = append(lines, "")
	}
	if hasCurrentValue {
		lines = append(lines, fmt.Sprintf("%s Leave empty to keep the current value.", palette.Color(term.Blue, "i")))
	} else if !definition.Required {
		lines = append(lines, fmt.Sprintf("%s Leave empty to skip.", palette.Color(term.Blue, "i")))
	}
	return strings.Join(lines, "\n")
}
