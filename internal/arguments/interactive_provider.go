package arguments

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type InteractiveProvider struct {
	input  io.Reader
	output io.Writer
}

func NewInteractiveProvider(in io.Reader, out io.Writer) *InteractiveProvider {
	return &InteractiveProvider{input: in, output: out}
}

func (p *InteractiveProvider) Provide(args []Arg) (map[string]string, error) {
	result := make(map[string]string)
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
			result[arg.Name] = value
		}
	}

	return result, nil
}

func (p *InteractiveProvider) Name() string {
	return "interactive"
}
