package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListGithubSources(t *testing.T) {
	t.Run("disallows sources with duplicate ids", func(t *testing.T) {
		sourcesJSON := `[
			{"repo":"example/repo","sha":"first-sha"},
			{"repo":"example/repo","sha":"second-sha"}
		]`
		sourcesFilePath := filepath.Join(t.TempDir(), "github_sources.json")
		require.NoError(t, os.WriteFile(sourcesFilePath, []byte(sourcesJSON), 0o600))

		_, err := ListGithubSources(sourcesFilePath)

		assert.EqualError(t, err, "duplicate source ID https://github.com/example/repo.git for example/repo@first-sha and example/repo@second-sha")
	})
}
