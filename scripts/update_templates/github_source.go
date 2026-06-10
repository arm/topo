package main

import (
	"encoding/json"
	"fmt"
	"io"
)

var sourcesJSON = `[
	{"repo": "Arm-Examples/topo-welcome", "sha": "8303e66db59a7a11e64877121f3db1b688d2011f"},
	{"repo": "Arm-Examples/topo-lightbulb-moment", "sha": "c2b2e4a672cda67832372f77aeb1d1f71beee9a7"},
	{"repo": "Arm-Examples/topo-cpu-ai-chat", "sha": "c06f343b9c753231714ac1cdbee7c7be5108e6b7"},
	{"repo": "Arm-Examples/topo-simd-visual-benchmark", "sha": "f0cd31621ce79b4643df7e9bdd8eff26c20b338c"}
]`

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
