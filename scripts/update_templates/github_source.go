package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const relativeSourcesPath = "scripts/update_templates/github_sources.json"

type GitHubSource struct {
	Repo string `json:"repo"`
	SHA  string `json:"sha"`
}

func (s GitHubSource) String() string {
	return fmt.Sprintf("%s@%s", s.Repo, s.SHA)
}

func (s GitHubSource) ID() TemplateSourceID {
	return TemplateSourceID(s.URL())
}

func (s GitHubSource) URL() string {
	return fmt.Sprintf("https://github.com/%s.git", s.Repo)
}

func ListGithubSources(path string) ([]GitHubSource, error) {
	sourcesFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer sourcesFile.Close() //nolint:errcheck // Closing a read-only file cannot affect catalog generation.

	var sources []GitHubSource
	if err := json.NewDecoder(sourcesFile).Decode(&sources); err != nil {
		return nil, fmt.Errorf("failed to decode sources: %w", err)
	}
	if err := validateUniqueGitHubSourceIDs(sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func GithubSourcesFilePath() (string, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, filepath.FromSlash(relativeSourcesPath)), nil
}

func validateUniqueGitHubSourceIDs(sources []GitHubSource) error {
	seen := make(map[TemplateSourceID]GitHubSource, len(sources))
	for _, source := range sources {
		id := source.ID()
		previous, exists := seen[id]
		if exists {
			return fmt.Errorf("duplicate source ID %s for %s and %s", id, previous, source)
		}
		seen[id] = source
	}
	return nil
}
