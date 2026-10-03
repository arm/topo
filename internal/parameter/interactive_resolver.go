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

func (r *InteractiveResolver) Resolve(parameters []Parameter) (Values, error) {
	values := Values{}
	if len(parameters) == 0 {
		return values, nil
	}
	scanner := bufio.NewScanner(r.input)
	palette := term.NewPaletteFor(r.output)

	for i, parameter := range parameters {
		prompt := fmt.Sprintf("%s\n", formatParameterPrompt(parameter, i+1, len(parameters), palette))
		if _, err := fmt.Fprint(r.output, prompt); err != nil {
			return nil, err
		}

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
				values[parameter.Name] = value
				break
			}
			if err := parameter.AssertSatisfiedBy(parameter.ExistingValue); err == nil {
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

func formatParameterPrompt(parameter Parameter, number, total int, palette term.Palette) string {
	progress := palette.Color(term.Dim, fmt.Sprintf("%d/%d", number, total))
	lines := []string{fmt.Sprintf("%s %s", progress, parameter.Name), ""}
	if description := strings.TrimSpace(parameter.Description); description != "" {
		lines = append(lines, fmt.Sprintf("    %s", strings.ReplaceAll(description, "\n", "\n    ")), "")
	}
	var metadata []string
	if parameter.ExistingValue != nil {
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Current:"), *parameter.ExistingValue))
	}
	if example := strings.TrimSpace(parameter.Example); example != "" {
		indentedExample := strings.ReplaceAll(example, "\n", "\n    ")
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Example:"), indentedExample))
	}
	if len(parameter.References) > 0 {
		metadata = append(metadata, "    "+palette.Color(term.Dim, "References:"))
		for _, reference := range parameter.References {
			pathLabel := palette.Color(term.Magenta, reference.Path+":")
			metadata = append(metadata, fmt.Sprintf("      %s %q", pathLabel, reference.Expression))
		}
	}
	if len(metadata) > 0 {
		lines = append(lines, metadata...)
		lines = append(lines, "")
	}
	if err := parameter.AssertSatisfiedBy(parameter.ExistingValue); err == nil {
		infoIcon := palette.Color(term.Blue, "i")
		if parameter.ExistingValue != nil {
			lines = append(lines, fmt.Sprintf("%s Leave empty to keep the current value.", infoIcon))
		} else {
			lines = append(lines, fmt.Sprintf("%s Leave empty to skip.", infoIcon))
		}
	}
	return strings.Join(lines, "\n")
}
