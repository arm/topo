package parameter

import (
	"fmt"
	"strings"
)

// CLIProvider resolves parameters from command-line key=value pairs.
// It validates that all provided keys match known parameter names.
type CLIProvider struct {
	input map[string]string
}

func NewCLIProvider(cliArgs []string) (*CLIProvider, error) {
	parsed := make(map[string]string)
	for _, arg := range cliArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter format: %s (expected PARAMETER=VALUE)", arg)
		}
		parsed[parts[0]] = parts[1]
	}
	return &CLIProvider{input: parsed}, nil
}

func (p *CLIProvider) Provide(definitions []Definition) ([]Provided, error) {
	var provided []Provided
	seen := make(map[string]bool, len(p.input))

	for _, definition := range definitions {
		if value, ok := p.input[definition.Name]; ok {
			provided = append(provided, Provided{Name: definition.Name, Value: value})
			seen[definition.Name] = true
		}
	}

	for key := range p.input {
		if !seen[key] {
			return nil, fmt.Errorf("unknown parameter: %s", key)
		}
	}

	return provided, nil
}
