package env

import (
	"bytes"
	"fmt"
	"os"

	"github.com/compose-spec/compose-go/v2/dotenv"
)

// Sources reports the winning assignment location, not the origin of interpolated fragments.
func Sources(paths []string, values map[string]string) (map[string]string, error) {
	sources := map[string]string{}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read parameter sources: %w", err)
		}
		entries, err := dotenv.ParseWithLookup(bytes.NewReader(content), func(key string) (string, bool) { value, ok := values[key]; return value, ok })
		if err != nil {
			return nil, fmt.Errorf("read sources from %s: %w", path, err)
		}
		for key := range entries {
			sources[key] = path
		}
	}
	for key := range values {
		if _, ok := os.LookupEnv(key); ok {
			sources[key] = "shell environment"
		}
	}
	return sources, nil
}
