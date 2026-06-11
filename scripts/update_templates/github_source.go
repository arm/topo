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

func (s GitHubSource) CloneURL() string {
	return fmt.Sprintf("https://github.com/%s.git", s.Repo)
}

func ListGitHubSources() ([]GitHubSource, error) {
	sourcesFile, err := openGitHubSources()
	if err != nil {
		return nil, err
	}
	defer sourcesFile.Close() //nolint:errcheck // Closing a read-only file cannot affect catalog generation.

	var sources []GitHubSource
	if err := json.NewDecoder(sourcesFile).Decode(&sources); err != nil {
		panic(fmt.Errorf("failed to decode sources: %w", err))
	}
	return sources, nil
}

func openGitHubSources() (*os.File, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	return os.Open(filepath.Join(repoRoot, filepath.FromSlash(relativeSourcesPath)))
}
