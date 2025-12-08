package source

import (
	"fmt"
	"strings"
)

func Parse(source string) (TemplateSource, error) {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid source format: %s (expected format: <type>:<value>, e.g., template:hello-world or git:https://github.com/user/repo.git)", source)
	}

	sourceType := parts[0]
	sourceValue := parts[1]

	if sourceValue == "" {
		return nil, fmt.Errorf("source value cannot be empty")
	}

	switch sourceType {
	case "template":
		return TemplateId(sourceValue), nil
	case "git":
		return parseGit(sourceValue), nil
	case "dir":
		return Dir{Path: sourceValue}, nil
	default:
		return nil, fmt.Errorf("unsupported source type: %s (supported: template:, git:, dir:)", sourceType)
	}
}

func parseGit(url string) Git {
	if idx := strings.LastIndex(url, "#"); idx != -1 {
		return Git{
			URL: url[:idx],
			Ref: url[idx+1:],
		}
	}

	return Git{URL: url, Ref: ""}
}
