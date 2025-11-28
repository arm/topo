package arguments

import (
	"fmt"
	"strings"
)

// CLIProvider resolves arguments from command-line key=value pairs.
// It validates that all provided keys match known argument names.
type CLIProvider struct {
	input map[string]string
}

func NewCLIProvider(cliArgs []string) (*CLIProvider, error) {
	parsed := make(map[string]string)
	for _, arg := range cliArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid argument format: %s (expected ARG=VALUE)", arg)
		}
		parsed[parts[0]] = parts[1]
	}
	return &CLIProvider{input: parsed}, nil
}

func (p *CLIProvider) Provide(args []Arg) ([]ResolvedArg, error) {
	var result []ResolvedArg
	seen := make(map[string]bool, len(p.input))

	for _, arg := range args {
		if value, ok := p.input[arg.Name]; ok {
			result = append(result, ResolvedArg{Name: arg.Name, Value: value})
			seen[arg.Name] = true
		}
	}

	for key := range p.input {
		if !seen[key] {
			return nil, fmt.Errorf("unknown argument: %s", key)
		}
	}

	return result, nil
}
