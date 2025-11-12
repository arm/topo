package source

import (
	"fmt"
	"strings"
)

func Parse(source string) (sourceType, sourceValue string, err error) {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid source format: %s (expected format: <type>:<value>, e.g., template:hello-world or git:https://github.com/user/repo.git)", source)
	}

	sourceType = parts[0]
	sourceValue = parts[1]

	if sourceValue == "" {
		return "", "", fmt.Errorf("source value cannot be empty")
	}

	return sourceType, sourceValue, nil
}
