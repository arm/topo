package parameter

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/output/term"
	// TODO consider if xterm should be entirely hidden behind topo/term
	xterm "golang.org/x/term"
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

	inFd := int(r.input.Fd())
	old, err := xterm.MakeRaw(inFd)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, xterm.Restore(inFd, old))
	}()

	reader := bufio.NewReader(r.input)
	palette := term.NewPaletteFor(r.output)

	for i, parameter := range parameters {
		var lastContent []string
		currentInput := ""
		if parameter.ExistingValue != nil {
			currentInput = *parameter.ExistingValue
		}

		// foreach keypress loop
		for {
			// re-measure terminal dimensions before nuking the previous content
			width, _, err := term.Dimensions(r.output)
			if err != nil {
				return nil, err
			}

			// clear last content
			if lastContentHeight := term.TextHeight(lastContent, width); lastContentHeight > 1 {
				fmt.Fprint(r.output, term.MoveUp(lastContentHeight-1))
			}
			fmt.Fprint(r.output, term.MoveToRowStart+term.ClearLine)

			// write new content
			content := formatParameterPrompt(parameter, currentInput, true, i+1, len(parameters), palette)
			fmt.Fprint(r.output, strings.Join(content, term.NewLine))
			lastContent = content

			// wait for keypress
			key, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}
			// TODO handle enter, backspace, and other control keys
			currentInput += string(key)
		}
	}
	return values, nil
}

func formatParameterPrompt(parameter Parameter, currentInput string, needsValue bool, number, total int, palette term.Palette) []string {
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
	lines = append(lines, fmt.Sprintf("%s %s", palette.Color(term.Magenta, ">"), currentInput))
	return lines
}
