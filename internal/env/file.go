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
	"unicode"

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

// UpdateFile applies assignments and removals; a nil update removes a variable and a non-nil update sets its value.
func UpdateFile(path string, changes map[string]*string, options EncodeOptions) error {
	if len(changes) == 0 {
		return nil
	}
	content, err := os.ReadFile(path)
	fileMissing := errors.Is(err, os.ErrNotExist)
	if err != nil && !fileMissing {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	updated, err := applyEnvChanges(string(content), encodeChanges(changes, options))
	if err != nil {
		return fmt.Errorf("failed to update env file: %w", err)
	}
	if fileMissing && updated == "" {
		return nil
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	return nil
}

func ToString(changes map[string]*string, options EncodeOptions) (string, error) {
	return applyEnvChanges("", encodeChanges(changes, options))
}

func encodeChanges(changes map[string]*string, options EncodeOptions) map[string]*string {
	encodedChanges := make(map[string]*string, len(changes))
	for name, value := range changes {
		if value == nil {
			encodedChanges[name] = nil
			continue
		}
		escaped := escapes.Replace(*value)
		if !options.PreserveInterpolation {
			escaped = strings.ReplaceAll(escaped, "$", "$$")
		}
		encodedChanges[name] = &escaped
	}
	return encodedChanges
}

func applyEnvChanges(content string, changes map[string]*string) (string, error) {
	remainingAssignments := maps.Clone(changes)
	for name, value := range remainingAssignments {
		if value == nil {
			delete(remainingAssignments, name)
		}
	}
	var result strings.Builder
	for len(content) > 0 {
		lineEnd := len(content)
		if index := strings.IndexByte(content, '\n'); index >= 0 {
			lineEnd = index + 1
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
		value, supplied := changes[header[1]]
		start := len(header[0])
		end, err := valueEnd(content, start)
		if err != nil {
			return "", err
		}
		if supplied && value == nil {
			content = content[assignmentEnd(content, end):]
			continue
		}
		result.WriteString(content[:start])
		if value, ok := changes[header[1]]; ok {
			result.WriteString("\"")
			result.WriteString(*value)
			result.WriteString("\"")
			delete(remainingAssignments, header[1])
		} else {
			result.WriteString(content[start:end])
		}
		content = content[end:]
	}
	if len(remainingAssignments) > 0 && result.Len() > 0 && !strings.HasSuffix(result.String(), "\n") {
		result.WriteByte('\n')
	}
	for _, name := range slices.Sorted(maps.Keys(remainingAssignments)) {
		result.WriteString(name)
		result.WriteString("=\"")
		result.WriteString(*remainingAssignments[name])
		result.WriteString("\"\n")
	}
	return result.String(), nil
}

func assignmentEnd(content string, valueEnd int) int {
	if index := strings.IndexByte(content[valueEnd:], '\n'); index >= 0 {
		return valueEnd + index + 1
	}
	return len(content)
}

func valueEnd(content string, valueStart int) (int, error) {
	if valueStart < len(content) && (content[valueStart] == '\'' || content[valueStart] == '"') {
		return quotedValueEnd(content, valueStart)
	}

	return unquotedValueEnd(content, valueStart)
}

func unquotedValueEnd(content string, valueStart int) (int, error) {
	valueEnd := valueStart
	for valueEnd < len(content) {
		character := content[valueEnd]
		if character == '\n' || character == '\r' {
			break
		}
		valueEnd++
	}
	// Remove inline comments
	value, _, _ := strings.Cut(content[valueStart:valueEnd], " #")
	value = strings.TrimRightFunc(value, unicode.IsSpace)
	return valueStart + len(value), nil
}

func quotedValueEnd(content string, openingQuoteIndex int) (int, error) {
	quote := content[openingQuoteIndex]
	for index := openingQuoteIndex + 1; index < len(content); index++ {
		// An escaped character cannot close the quoted value.
		if content[index] == '\\' {
			index++
			continue
		}
		if content[index] == quote {
			return index + 1, nil
		}
	}
	return 0, errors.New("unterminated quoted env value")
}
