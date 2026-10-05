package parameter

import (
	"fmt"
	"strings"
)

// CLIResolver resolves assignments and removals supplied on the command line.
// It validates that all provided keys match known parameter names.
type CLIResolver struct {
	changes Changes
}

func NewCLIResolver(assignments []string, removals []string) (*CLIResolver, error) {
	changes := make(Changes)
	for _, arg := range assignments {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid parameter format: %s (expected PARAMETER=VALUE)", arg)
		}
		changes[parts[0]] = new(parts[1])
	}
	for _, name := range removals {
		if value, exists := changes[name]; exists && value != nil {
			return nil, fmt.Errorf("cannot both set and unset parameter: %s", name)
		}
		changes[name] = nil
	}
	return &CLIResolver{changes: changes}, nil
}

func (r *CLIResolver) Resolve(parameters []Parameter) (Changes, error) {
	changes := Changes{}
	seen := make(map[string]bool, len(r.changes))

	for _, parameter := range parameters {
		name := parameter.Name
		if value, ok := r.changes[name]; ok {
			changes[name] = value
			seen[name] = true
		}
	}

	for key := range r.changes {
		if !seen[key] {
			return nil, fmt.Errorf("unknown parameter: %s", key)
		}
	}

	return changes, nil
}
