package parameter

import (
	"fmt"
	"strings"

	"github.com/arm/topo/internal/output/term"
	"github.com/clipperhouse/displaywidth"
	"github.com/compose-spec/compose-go/v2/template"
)

// InteractiveResolver resolves parameter definitions to changes by prompting for values.
type InteractiveResolver struct {
	readPrompt func(term.Prompt) (string, error)
	palette    term.Palette
}

func NewInteractiveResolver(readPrompt func(term.Prompt) (string, error), palette term.Palette) *InteractiveResolver {
	return &InteractiveResolver{readPrompt: readPrompt, palette: palette}
}

func (r *InteractiveResolver) Resolve(parameters []Parameter) (Changes, error) {
	changes := Changes{}

	for i, parameter := range parameters {
		initial := ""
		if parameter.ExistingValue != nil {
			initial = *parameter.ExistingValue
		}

		value, err := r.readPrompt(term.Prompt{
			Validate: func(input string) bool {
				return parameter.AssertSatisfiedBy(contentfulStringOrNil(input)) == nil
			},
			Content: func(input string) []string {
				return formatParameterPromptContent(parameter, input, i+1, len(parameters), r.palette)
			},
			Initial: initial,
			Prefix:  r.palette.Color(term.Magenta, ">") + " ",
		})
		if err != nil {
			return nil, err
		}

		if value == "" && parameter.ExistingValue == nil {
			continue
		}
		if parameter.ExistingValue != nil && value == *parameter.ExistingValue {
			continue
		}
		changes[parameter.Name] = contentfulStringOrNil(value)
	}
	return changes, nil
}

func formatParameterPromptContent(parameter Parameter, currentInput string, number, total int, palette term.Palette) []string {
	progress := palette.Color(term.Dim, fmt.Sprintf("%d/%d", number, total))
	lines := []string{fmt.Sprintf("%s %s", progress, parameter.Name), ""}
	if description := strings.TrimSpace(parameter.Description); description != "" {
		for line := range strings.SplitSeq(description, "\n") {
			lines = append(lines, fmt.Sprintf("    %s", line))
		}
	}
	var metadata []string
	if example := strings.TrimSpace(parameter.Example); example != "" {
		metadata = append(metadata, fmt.Sprintf("    %s %q", palette.Color(term.Dim, "Example:"), example))
	}
	if len(parameter.References) > 0 {
		metadata = append(metadata, "    "+palette.Color(term.Dim, "References:"))
		pathWidth := 0
		for _, reference := range parameter.References {
			pathWidth = max(pathWidth, displaywidth.String(reference.Path+":"))
		}
		lookup := func(name string) (string, bool) {
			if name == parameter.Name && currentInput != "" {
				return currentInput, true
			}
			return "", false
		}
		for _, reference := range parameter.References {
			pathLabel := palette.Color(term.Magenta, reference.Path+":")
			value, err := template.SubstituteWithOptions(reference.Expression, lookup, template.WithoutLogging)
			preview := fmt.Sprintf("%q", value)
			if err != nil {
				preview = palette.Color(term.Red, "✗") + " " + err.Error()
			}
			padding := strings.Repeat(" ", pathWidth-displaywidth.String(reference.Path+":"))
			metadata = append(metadata, fmt.Sprintf("      %s%s %s", pathLabel, padding, preview))
			raw := palette.Color(term.Dim, fmt.Sprintf("%s", reference.Expression))
			metadata = append(metadata, strings.Repeat(" ", 6+pathWidth+1)+raw)
		}
	}
	if len(metadata) > 0 {
		lines = append(lines, metadata...)
		lines = append(lines, "")
	}

	if err := parameter.AssertSatisfiedBy(contentfulStringOrNil(currentInput)); err != nil {
		lines = append(lines, fmt.Sprintf("%s %s", palette.Color(term.Red, "✗"), err.Error()))
	} else {
		lines = append(lines, fmt.Sprintf("%s Press enter to continue.", palette.Color(term.Green, "✓")))
	}
	return lines
}

func contentfulStringOrNil(input string) *string {
	if input == "" {
		return nil
	}
	return &input
}
