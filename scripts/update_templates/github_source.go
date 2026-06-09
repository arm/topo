package main

import (
	"encoding/json"
	"fmt"
	"io"
)

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

func ListGitHubSources(jsonData io.Reader) []GitHubSource {
	var sources []GitHubSource
	if err := json.NewDecoder(jsonData).Decode(&sources); err != nil {
		panic(fmt.Errorf("failed to decode sources: %w", err))
	}
	return sources
}
