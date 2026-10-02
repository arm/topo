package parameter

import (
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/output/logger"
	"github.com/arm/topo/internal/output/term"
)

// InteractiveResolver resolves parameter definitions to values by prompting via stdin/stdout.
type InteractiveResolver struct {
	input  *os.File
	output *os.File
}

func NewInteractiveResolver(in *os.File, out *os.File) *InteractiveResolver {
	return &InteractiveResolver{input: in, output: out}
}

func (r *InteractiveResolver) Resolve(parameters []Parameter) (_ Values, err error) {
	values := Values{}
	if len(parameters) == 0 {
		return values, nil
	}
	if !term.IsTerminal(r.input) || !term.IsTerminal(r.output) {
		panic("internal error: interactive resolver not running in an interactive terminal")
	}

	palette := term.NewPaletteFor(r.output)

	for i, parameter := range parameters {
		initial := ""
		if parameter.ExistingValue != nil {
			initial = *parameter.ExistingValue
		}
		needsValue := parameter.needsValue()

		value, err := term.ReadPrompt(r.input, r.output, term.Prompt{
			Validate: func(input string) error {
				if input == "" && needsValue {
					return fmt.Errorf("%s A value is required.", palette.Color(term.Red, "✗"))
				}
				return nil
			},
			Content: func(input string, validationErr error) []string {
				return formatParameterPrompt(parameter, input, validationErr, needsValue, i+1, len(parameters), palette)
			},
			Initial: initial,
		})
		logger.Info(fmt.Sprintf("%s %v", value, err))
	}
	return values, nil
}

func formatParameterPrompt(parameter Parameter, currentInput string, validationErr error, needsValue bool, number, total int, palette term.Palette) []string {
	progress := palette.Color(term.Dim, fmt.Sprintf("%d/%d", number, total))
	lines := []string{fmt.Sprintf("%s %s", progress, parameter.Name), ""}
	if description := strings.TrimSpace(parameter.Description); description != "" {
		for _, line := range strings.Split(description, "\n") {
			lines = append(lines, fmt.Sprintf("    %s", line))
		}
	}
	var metadata []string
	if parameter.ExistingValue != nil {
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Current:"), *parameter.ExistingValue))
	}
	if example := strings.TrimSpace(parameter.Example); example != "" {
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Example:"), example))
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
	if parameter.ExistingValue != nil {
		lines = append(lines, fmt.Sprintf("%s Leave empty to keep the current value.", palette.Color(term.Blue, "i")))
	} else if !needsValue {
		lines = append(lines, fmt.Sprintf("%s Leave empty to skip.", palette.Color(term.Blue, "i")))
	}
	if validationErr != nil {
		lines = append(lines, fmt.Sprintf("    %s", validationErr.Error()))
	}
	lines = append(lines, fmt.Sprintf("%s %s", palette.Color(term.Magenta, ">"), currentInput))
	return lines
}
