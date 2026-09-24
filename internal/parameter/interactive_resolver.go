package parameter

import (
	"bufio"
	"fmt"
	"io"
	"strings"
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
	scanner := bufio.NewScanner(r.input)

	for i, definition := range definitions {
		if i != 0 {
			_, err := fmt.Fprintf(r.output, "\n")
			if err != nil {
				return nil, err
			}
		}
		_, err := fmt.Fprintf(r.output, "Provide: %s\n", definition.Description)
		if err != nil {
			return nil, err
		}

		if definition.Example != "" {
			_, err := fmt.Fprintf(r.output, "Example: %s\n", definition.Example)
			if err != nil {
				return nil, err
			}
		}

		currentValue, hasCurrentValue := currentValues[definition.Name]
		if hasCurrentValue {
			_, err := fmt.Fprintf(r.output, "Current: %q\n", currentValue)
			if err != nil {
				return nil, err
			}
		}

		label := "optional"
		if definition.Required {
			label = "required"
		}
		if hasCurrentValue {
			label += ", leave blank to keep current"
		}

		_, err = fmt.Fprintf(r.output, "%s (%s)> ", definition.Name, label)
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
			values[definition.Name] = value
		}
	}

	return values, nil
}
