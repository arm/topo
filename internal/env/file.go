package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultFilename = ".env.topo"

func ResolveFiles(root string, files []string, skipMissing bool) ([]string, error) {
	var paths []string
	for _, filename := range files {
		path := filepath.Join(root, filename)
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		} else if errors.Is(err, os.ErrNotExist) {
			if skipMissing {
				continue
			}
			return nil, fmt.Errorf("env file %q does not exist", path)
		} else {
			return nil, fmt.Errorf("failed to check env file: %w", err)
		}
	}
	return paths, nil
}
