package env

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
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

func UpdateFile(path string, values map[string]string) error {
	merged, err := dotenv.ReadFile(path, nil)
	if errors.Is(err, os.ErrNotExist) {
		merged = make(map[string]string)
	} else if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	maps.Copy(merged, values)
	content, err := encodeFile(merged)
	if err != nil {
		return fmt.Errorf("failed to encode env file content: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	return nil
}

func WriteFile(output io.Writer, values map[string]string) error {
	content, err := encodeFile(values)
	if err != nil {
		return fmt.Errorf("failed to encode env file content: %w", err)
	}
	if _, err := io.WriteString(output, content); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}
	return nil
}

func encodeFile(values map[string]string) (string, error) {
	var content strings.Builder
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if name == "" {
			return "", errors.New("env parameter name must not be empty")
		}
		content.WriteString(name)
		content.WriteString("=\"")
		content.WriteString(escapes.Replace(values[name]))
		content.WriteString("\"\n")
	}

	return content.String(), nil
}
