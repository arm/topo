package arguments

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/arm-debug/topo-cli/internal/service"
)

type InteractiveProvider struct {
	input  io.Reader
	output io.Writer
}

func NewInteractiveProvider(in io.Reader, out io.Writer) *InteractiveProvider {
	return &InteractiveProvider{input: in, output: out}
}

func (p *InteractiveProvider) Provide(specs []service.ArgSpec) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(p.input)

	for _, spec := range specs {
		fmt.Fprintf(p.output, "\n%s\n", spec.Description)

		if spec.Example != "" {
			fmt.Fprintf(p.output, "Example: %s\n", spec.Example)
		}

		requiredLabel := ""
		if spec.Required {
			requiredLabel = " (required)"
		}
		fmt.Fprintf(p.output, "%s%s> ", spec.Name, requiredLabel)

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, err
			}
			break
		}

		value := strings.TrimSpace(scanner.Text())
		if value != "" {
			result[spec.Name] = value
		}
	}

	return result, nil
}

func (p *InteractiveProvider) Name() string {
	return "interactive"
}
