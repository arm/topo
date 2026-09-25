package env

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/compose-spec/compose-go/v2/dotenv"
)

const DefaultFilename = ".env.topo"

var DefaultFilenames = []string{".env", DefaultFilename}

var escapes = strings.NewReplacer(
	"\\", "\\\\",
	"\"", "\\\"",
	"\n", "\\n",
	"\r", "\\r",
	"\t", "\\t",
)

func ResolveFiles(root string, paths []string, skipMissing bool) ([]string, error) {
	var resolvedPaths []string
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		if _, err := os.Stat(path); err == nil {
			resolvedPaths = append(resolvedPaths, path)
		} else if errors.Is(err, os.ErrNotExist) {
			if skipMissing {
				continue
			}
			return nil, fmt.Errorf("env file %q does not exist", path)
		} else {
			return nil, fmt.Errorf("failed to check env file: %w", err)
		}
	}
	return resolvedPaths, nil
}

func ReadFiles(paths []string) (map[string]string, error) {
	content, err := dotenv.GetEnvFromFile(nil, paths)
	if err != nil {
		return nil, fmt.Errorf("failed to read env files: %w", err)
	}
	return content, nil
}

var assignmentHeader = regexp.MustCompile(`^[ \t]*(?:export[ \t]+)?([\pL\pN_.\[\]-]+)[ \t]*[=:][ \t]*`)

type EncodeOptions struct {
	PreserveInterpolation bool
}

func UpdateFile(path string, updates map[string]string, options EncodeOptions) error {
	if len(updates) == 0 {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	updated, err := applyFileUpdates(string(content), encodeValues(updates, options))
	if err != nil {
		return fmt.Errorf("failed to update env file: %w", err)
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	return nil
}

func ToString(values map[string]string, options EncodeOptions) (string, error) {
	return applyFileUpdates("", encodeValues(values, options))
}

func encodeValues(values map[string]string, options EncodeOptions) map[string]string {
	encoded := make(map[string]string, len(values))
	for name, value := range values {
		encoded[name] = escapes.Replace(value)
		if !options.PreserveInterpolation {
			encoded[name] = strings.ReplaceAll(encoded[name], "$", "$$")
		}
	}
	return encoded
}

func applyFileUpdates(content string, updates map[string]string) (string, error) {
	remaining := maps.Clone(updates)
	var result strings.Builder
	for len(content) > 0 {
		lineEnd := strings.IndexByte(content, '\n')
		if lineEnd < 0 {
			lineEnd = len(content)
		} else {
			lineEnd++
		}
		line := content[:lineEnd]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result.WriteString(line)
			content = content[lineEnd:]
			continue
		}
		header := assignmentHeader.FindStringSubmatch(content)
		if header == nil {
			return "", errors.New("unsupported env assignment syntax")
		}
		start := len(header[0])
		end, err := valueEnd(content, start)
		if err != nil {
			return "", err
		}
		result.WriteString(content[:start])
		if value, ok := updates[header[1]]; ok {
			result.WriteString("\"")
			result.WriteString(value)
			result.WriteString("\"")
			delete(remaining, header[1])
		} else {
			result.WriteString(content[start:end])
		}
		content = content[end:]
	}
	if len(remaining) > 0 && result.Len() > 0 && !strings.HasSuffix(result.String(), "\n") {
		result.WriteByte('\n')
	}
	for _, name := range slices.Sorted(maps.Keys(remaining)) {
		result.WriteString(name)
		result.WriteString("=\"")
		result.WriteString(remaining[name])
		result.WriteString("\"\n")
	}
	return result.String(), nil
}

func valueEnd(content string, start int) (int, error) {
	if start < len(content) && (content[start] == '\'' || content[start] == '"') {
		quote := content[start]
		for i := start + 1; i < len(content); i++ {
			if content[i] == '\\' {
				i++
				continue
			}
			if content[i] == quote {
				return i + 1, nil
			}
		}
		return 0, errors.New("unterminated quoted env value")
	}
	end := start
	for end < len(content) && content[end] != '\n' && content[end] != '\r' {
		if content[end] == '#' && (end == start || content[end-1] == ' ' || content[end-1] == '\t') {
			break
		}
		end++
	}
	return start + len(strings.TrimRight(content[start:end], " \t")), nil
}
