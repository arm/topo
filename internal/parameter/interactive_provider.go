package parameter

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// InteractiveProvider prompts the user for each parameter via stdin/stdout.
type InteractiveProvider struct {
	input  io.Reader
	output io.Writer
}

func NewInteractiveProvider(in io.Reader, out io.Writer) *InteractiveProvider {
	return &InteractiveProvider{input: in, output: out}
}

func (p *InteractiveProvider) Provide(definitions []Definition) ([]Provided, error) {
	var provided []Provided
	scanner := bufio.NewScanner(p.input)

	for i, definition := range definitions {
		if i != 0 {
			_, err := fmt.Fprintf(p.output, "\n")
			if err != nil {
				return nil, err
			}
		}
		_, err := fmt.Fprintf(p.output, "Provide: %s\n", definition.Description)
		if err != nil {
			return nil, err
		}

		if definition.Example != "" {
			_, err := fmt.Fprintf(p.output, "Example: %s\n", definition.Example)
			if err != nil {
				return nil, err
			}
		}

		if len(definition.CurrentValues) > 0 {
			_, err := fmt.Fprintf(p.output, "Current: %s\n", formatCurrentValues(definition.CurrentValues))
			if err != nil {
				return nil, err
			}
		}

		label := "optional"
		if definition.Required {
			label = "required"
		}
		if len(definition.CurrentValues) > 0 {
			label += ", leave blank to keep current"
		}

		_, err = fmt.Fprintf(p.output, "%s (%s)> ", definition.Name, label)
		if err != nil {
			return nil, err
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, err
			}
			break
		}

		value := strings.TrimSpace(scanner.Text())
		if value != "" {
			provided = append(provided, Provided{Name: definition.Name, Value: value})
		}
	}

	return provided, nil
}
