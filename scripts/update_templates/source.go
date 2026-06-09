package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
)

// TODO: sha should become a hash
var sourcesJSON = `[
	{"repo": "Arm-Examples/topo-welcome", "sha": "main"},
	{"repo": "Arm-Examples/topo-lightbulb-moment", "sha": "main"},
	{"repo": "Arm-Examples/topo-cpu-ai-chat", "sha": "main"},
	{"repo": "Arm-Examples/topo-simd-visual-benchmark", "sha": "main"}
]`

type GitHubSource struct {
	Repo string `json:"repo"`
	SHA  string `json:"sha"`
}

func (s GitHubSource) String() string {
	return fmt.Sprintf("%s@%s", s.Repo, s.SHA)
}

func (s GitHubSource) URL() string {
	return fmt.Sprintf("https://github.com/%s.git", s.Repo)
}

func (s GitHubSource) FileURL(repoFilePath string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   path.Join("repos", s.Repo, "contents", repoFilePath),
	}
	q := u.Query()
	q.Set("ref", s.SHA)
	u.RawQuery = q.Encode()
	return u.String()
}

func ListSources(jsonData io.Reader) []GitHubSource {
	var sources []GitHubSource
	if err := json.NewDecoder(jsonData).Decode(&sources); err != nil {
		panic(fmt.Errorf("failed to decode sources: %w", err))
	}
	return sources
}
