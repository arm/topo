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
	Id          string   `json:"id"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	Url         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func GetTemplateRepo(id string) (*Repo, error) {
	return GetRepo(id, TemplatesJSON)
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

func GetRepo(id string, b []byte) (*Repo, error) {
	repos, err := ParseRepos(b)
	if err != nil {
		return nil, err
	}
	for i := range repos {
		if repos[i].Id == id {
			return &repos[i], nil
		}
	}
	return nil, fmt.Errorf("repo with id %q not found", id)
}
