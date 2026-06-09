package main

import (
	"encoding/json"
	"fmt"
	"io"
)

// TODO: sha should become a hash
var sourcesJSON = `[
	{"repo": "Arm-Examples/topo-welcome", "sha": "main"},
	{"repo": "Arm-Examples/topo-lightbulb-moment", "sha": "main"},
	{"repo": "Arm-Examples/topo-cpu-ai-chat", "sha": "main"},
	{"repo": "Arm-Examples/topo-simd-visual-benchmark", "sha": "main"}
]`

type Source struct {
	Repo string `json:"repo"`
	SHA  string `json:"sha"`
}

func (s Source) String() string {
	return fmt.Sprintf("%s@%s", s.Repo, s.SHA)
}

func ListSources(jsonData io.Reader) []Source {
	var sources []Source
	if err := json.NewDecoder(jsonData).Decode(&sources); err != nil {
		panic(fmt.Errorf("failed to decode sources: %w", err))
	}
	return sources
}
