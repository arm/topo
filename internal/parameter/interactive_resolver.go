package parameter

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/arm/topo/pkg/plugin/suggestions"
)

// InteractiveResolver resolves parameter definitions to changes by prompting for values.
type InteractiveResolver struct {
	Context            context.Context
	Providers          []suggestions.Provider
	TargetObservations observations.Tree
	readPrompt         func(term.Prompt) (string, error)
	palette            term.Palette
	prefillResolver    Resolver
}

func NewInteractiveResolverForTerminal(in, out *os.File, prefillResolver Resolver) *InteractiveResolver {
	readPrompt := func(prompt term.Prompt) (string, error) {
		return term.ReadPrompt(in, out, prompt)
	}
	return NewInteractiveResolver(readPrompt, term.NewPaletteFor(out), prefillResolver)
}

func NewInteractiveResolver(
	readPrompt func(term.Prompt) (string, error), palette term.Palette, prefillResolver Resolver,
) *InteractiveResolver {
	return &InteractiveResolver{readPrompt: readPrompt, palette: palette, prefillResolver: prefillResolver}
}

func (r *InteractiveResolver) Resolve(parameters []Parameter) (Changes, error) {
	var detected Changes
	if r.prefillResolver != nil {
		var err error
		detected, err = r.prefillResolver.Resolve(parameters)
		if err != nil {
			return nil, err
		}
	}

	changes := Changes{}
	for i, parameter := range parameters {
		detectedValue := detected[parameter.Name]
		initial := ""
		if parameter.ExistingValue != nil {
			initial = *parameter.ExistingValue
		} else if detectedValue != nil {
			initial = *detectedValue
		}

		actions, labels, err := r.parameterActions(parameter)
		if err != nil {
			return nil, err
		}
		valueFromInput := func(input string) *string {
			if detectedValue != nil && input == *detectedValue {
				return detectedValue
			}
			return contentfulStringOrNil(input)
		}
		validate := func(input string) bool {
			return parameter.AssertSatisfiedBy(valueFromInput(input)) == nil
		}
		value, err := r.readPrompt(term.Prompt{
			Actions:  actions,
			Validate: validate,
			Content:  formatParameterPromptContent(parameter, i+1, len(parameters), detectedValue, r.palette),
			Hint: func(input string, showValidationError bool) string {
				offerDetectedValue := detectedValue != nil && input != *detectedValue
				return formatParameterPromptHint(
					parameter, validate(input), showValidationError, offerDetectedValue, labels, r.palette,
				)
			},
			Initial:      initial,
			Prefix:       "   " + r.palette.Color(term.Magenta, ">") + " ",
			RestoreValue: detectedValue,
		})
		if err != nil {
			return nil, err
		}

		if parameter.ExistingValue != nil && value == *parameter.ExistingValue {
			continue
		}
		if parameter.ExistingValue == nil && detectedValue == nil && value == "" {
			continue
		}
		changes[parameter.Name] = valueFromInput(value)
	}
	return changes, nil
}

func formatParameterPromptContent(
	parameter Parameter, number, total int, detectedValue *string, palette term.Palette,
) []string {
	progress := palette.Color(term.Dim, fmt.Sprintf("%d/%d", number, total))
	lines := []string{fmt.Sprintf("   %s %s", progress, parameter.Name), ""}
	if description := strings.TrimSpace(parameter.Description); description != "" {
		for line := range strings.SplitSeq(description, "\n") {
			lines = append(lines, fmt.Sprintf("       %s", line))
		}
	}
	if example := strings.TrimSpace(parameter.Example); example != "" {
		lines = append(lines, fmt.Sprintf("       %s %q", palette.Color(term.Dim, "Example:"), example))
	}
	if len(parameter.References) > 0 {
		lines = append(lines, "       "+palette.Color(term.Dim, "References:"))
		for _, reference := range parameter.References {
			pathLabel := palette.Color(term.Magenta, reference.Path+":")
			lines = append(lines, fmt.Sprintf("         %s %q", pathLabel, reference.Expression))
		}
	}
	if len(lines) > 2 {
		lines = append(lines, "")
	}
	if detectedValue != nil {
		lines = append(lines, "   "+palette.Color(term.Blue, "i")+" Detected on target: "+*detectedValue)
	}
	if number > 1 {
		lines = append([]string{""}, lines...)
	}
	return lines
}

func formatParameterPromptHint(
	parameter Parameter, canSubmit, showValidationError, offerDetectedValue bool, actionHints string, palette term.Palette,
) string {
	message := "Enter a value to continue"
	if canSubmit {
		message = "Press Enter to continue"
	}
	message = palette.Color(term.Dim, message)
	if showValidationError {
		message = fmt.Sprintf("%s %s is required, enter a value to continue", palette.Color(term.Red, "✗"), parameter.Name)
	}
	if offerDetectedValue {
		message += palette.Color(term.Dim, " · Ctrl+R to use detected value")
	}
	return "   " + message + actionHints + palette.Color(term.Dim, " · Ctrl+C to cancel")
}

func contentfulStringOrNil(input string) *string {
	if input == "" {
		return nil
	}
	return &input
}
