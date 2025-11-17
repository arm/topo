package arguments

import (
	"fmt"
	"strings"
)

type CLIProvider map[string]string

func NewCLIProvider(cliArgs []string) (CLIProvider, error) {
	parsed := make(map[string]string)
	for _, arg := range cliArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid argument format: %s (expected ARG=VALUE)", arg)
		}
		parsed[parts[0]] = parts[1]
	}
	return CLIProvider(parsed), nil
}

func (p CLIProvider) Provide(args []Arg) (map[string]string, error) {
	for key := range p {
		found := false
		for _, arg := range args {
			if arg.Name == key {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown argument: %s", key)
		}
	}
	return p, nil
}

func (p CLIProvider) Name() string {
	return "cli"
}
