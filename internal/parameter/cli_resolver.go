package parameter

import (
	"fmt"
	"strings"
)

// CLIResolver resolves parameter definitions to values from command-line key=value pairs.
// It validates that all provided keys match known parameter names.
type CLIResolver struct {
	input map[string]string
}

func NewCLIResolver(cliArgs []string) (*CLIResolver, error) {
	parsed := make(map[string]string)
	for _, arg := range cliArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter format: %s (expected PARAMETER=VALUE)", arg)
		}
		parsed[parts[0]] = parts[1]
	}
	return &CLIResolver{input: parsed}, nil
}

func (r *CLIResolver) Resolve(definitions []Definition) (Values, error) {
	values := Values{}
	seen := make(map[string]bool, len(r.input))

	for _, definition := range definitions {
		if value, ok := r.input[definition.Name]; ok {
			values[definition.Name] = value
			seen[definition.Name] = true
		}
	}

	for key := range r.input {
		if !seen[key] {
			return nil, fmt.Errorf("unknown parameter: %s", key)
		}
	}

	return values, nil
}
