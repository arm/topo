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

	for _, arg := range args {
		fmt.Fprintf(p.output, "\n%s\n", arg.Description)

		if arg.Example != "" {
			fmt.Fprintf(p.output, "Example: %s\n", arg.Example)
		}

		requiredLabel := ""
		if arg.Required {
			requiredLabel = " (required)"
		}
		fmt.Fprintf(p.output, "%s%s> ", arg.Name, requiredLabel)

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
