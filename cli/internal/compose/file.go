package compose

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/cli/internal/output/logger"
)

var errFileNotFound = errors.New("compose file not found")

var defaultFileNames = []string{"compose.yaml", "compose.yml"}

// DefaultFileName returns the first filename used when resolving a default compose file.
func DefaultFileName() string {
	return defaultFileNames[0]
}

// RequireFile returns the path when it exists and is not a directory.
func RequireFile(composeFile string) (string, error) {
	info, err := os.Stat(composeFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", errFileNotFound, composeFile)
		}
		return "", fmt.Errorf("failed to access compose file %s: %w", composeFile, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("compose file path is a directory: %s", composeFile)
	}
	return composeFile, nil
}

// FindDefaultFile returns the first existing default compose file.
func FindDefaultFile() (string, error) {
	type defaultFileCandidate struct {
		name string
		info os.FileInfo
	}
	candidates := []defaultFileCandidate{}

	for _, fileName := range defaultFileNames {
		info, err := os.Stat(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("failed to access compose file %s: %w", fileName, err)
		}

		candidates = append(candidates, defaultFileCandidate{
			name: fileName,
			info: info,
		})
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf(
			"compose file not found in current working directory: looking for %s",
			strings.Join(defaultFileNames, " or "),
		)
	}

	defaultCandidate := candidates[0]
	if len(candidates) > 1 {
		logger.Warn(fmt.Sprintf(
			"found multiple compose files: %s; using %s",
			strings.Join(defaultFileNames, ", "),
			defaultCandidate.name,
		))
	}
	if defaultCandidate.info.IsDir() {
		return "", fmt.Errorf("compose file path is a directory: %s", defaultCandidate.name)
	}
	return defaultCandidate.name, nil
}
