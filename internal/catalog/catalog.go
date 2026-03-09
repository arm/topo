package catalog

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/templates.json
var TemplatesJSON []byte

type Repo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	MinRAMKb    int64    `json:"min_ram_kb,omitempty"`
	URL         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func GetTemplateRepo(name string) (*Repo, error) {
	return GetRepo(name, TemplatesJSON)
}

func ParseRepos(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetRepo(name string, b []byte) (*Repo, error) {
	repos, err := ParseRepos(b)
	if err != nil {
		return nil, err
	}
	for i := range repos {
		if repos[i].Name == name {
			return &repos[i], nil
		}
	}
	return nil, fmt.Errorf("repo with name %q not found", name)
}
