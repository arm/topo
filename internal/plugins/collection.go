package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arm/topo/pkg/plugin/observations"
	"github.com/arm/topo/pkg/plugin/suggestions"
	"github.com/arm/topo/pkg/plugin/transport"
)

func Directory() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(directory, "topo", "plugins"), nil
}

type Collection struct {
	cancel  context.CancelFunc
	plugins []*process
}

func LoadAll(ctx context.Context, directory string) (result *Collection, failures []error, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	paths, failures, err := executables(directory)
	if err != nil {
		return nil, failures, err
	}
	ctx, cancel := context.WithCancel(ctx)
	collection := &Collection{cancel: cancel}
	defer func() {
		if err != nil {
			collection.Close()
		}
	}()
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, failures, err
		}
		loaded, err := load(ctx, path)
		if err != nil {
			if ctx.Err() != nil {
				return nil, failures, ctx.Err()
			}
			failures = append(failures, err)
			continue
		}
		collection.plugins = append(collection.plugins, loaded)
	}
	if err := ctx.Err(); err != nil {
		return nil, failures, err
	}
	failures = append(failures, collection.rejectDuplicateIdentities()...)
	return collection, failures, nil
}

// Reject every member of a collision rather than selecting a provider by filename.
func (c *Collection) rejectDuplicateIdentities() []error {
	counts := make(map[string]int, len(c.plugins))
	for _, loaded := range c.plugins {
		counts[loaded.descriptor.ID]++
	}
	unique := make([]*process, 0, len(c.plugins))
	var failures []error
	for _, loaded := range c.plugins {
		if counts[loaded.descriptor.ID] > 1 {
			loaded.Close()
			failures = append(failures, fmt.Errorf("plugin %q: duplicate plugin identity %q",
				loaded.path, loaded.descriptor.ID))
			continue
		}
		unique = append(unique, loaded)
	}
	c.plugins = unique
	return failures
}

func (c *Collection) Close() {
	for _, loaded := range c.plugins {
		loaded.Close()
	}
	c.cancel()
}

func (c *Collection) Observations() []observations.Provider {
	var selected []observations.Provider
	for _, loaded := range c.plugins {
		if loaded.descriptor.Capability == transport.Observations {
			selected = append(selected, observations.NewIdentityPrefixer(loaded.descriptor.ID, loaded))
		}
	}
	return selected
}

func (c *Collection) Suggestions() []suggestions.Provider {
	var selected []suggestions.Provider
	for _, loaded := range c.plugins {
		if loaded.descriptor.Capability == transport.Suggestions {
			selected = append(selected, loaded)
		}
	}
	return selected
}

func executables(directory string) ([]string, []error, error) {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("reading plugin directory: %w", err)
	}
	var paths []string
	var failures []error
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "topo-plugin-") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("checking plugin %q: %w", path, err))
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if runtime.GOOS == "windows" {
			if !strings.HasSuffix(strings.ToLower(entry.Name()), ".exe") {
				continue
			}
		} else if info.Mode().Perm()&0o111 == 0 {
			continue
		}
		paths = append(paths, path)
	}
	return paths, failures, nil
}
