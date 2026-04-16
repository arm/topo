package arguments

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// InteractiveProvider prompts the user for each argument via stdin/stdout.
type InteractiveProvider struct {
	input  io.Reader
	output io.Writer
}

func NewInteractiveProvider(in io.Reader, out io.Writer) *InteractiveProvider {
	return &InteractiveProvider{input: in, output: out}
}

func (p *InteractiveProvider) Provide(args []Arg) ([]ResolvedArg, error) {
	var result []ResolvedArg
	scanner := bufio.NewScanner(p.input)

	for i, arg := range args {
		if i != 0 {
			fmt.Printf("\n")
		}
		_, err := fmt.Fprintf(p.output, "Provide: %s\n", arg.Description)
		if err != nil {
			return nil, err
		}

		if arg.Example != "" {
			_, err := fmt.Fprintf(p.output, "Example: %s\n", arg.Example)
			if err != nil {
				return nil, err
			}
		}

		if arg.Default != "" {
			_, err := fmt.Fprintf(p.output, "Default: %s\n", arg.Default)
			if err != nil {
				return nil, err
			}
		}

		requiredLabel := ""
		if arg.Required {
			requiredLabel = " (required)"
		}

		_, err = fmt.Fprintf(p.output, "%s%s> ", arg.Name, requiredLabel)
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
			result = append(result, ResolvedArg{Name: arg.Name, Value: value})
		}
	}

	return result, nil
}
