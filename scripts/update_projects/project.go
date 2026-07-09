package main

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type ProjectSourceID string

type Project struct {
	XTopo
	URL string `json:"url"`
	Ref string `json:"ref"`
}

type XTopo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Features    []string       `json:"features"`
	Args        map[string]Arg `json:"args,omitempty"`
}

type Arg struct {
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Default     string         `json:"default,omitempty"`
	Example     string         `json:"example,omitempty"`
	Hints       map[string]any `json:"hints,omitempty"`
}

func NewProject(source GitHubSource, composeFile io.Reader) (Project, error) {
	type composeDocument struct {
		XTopo XTopo `yaml:"x-topo"`
	}

	var parsed composeDocument
	decoder := yaml.NewDecoder(composeFile)
	if err := decoder.Decode(&parsed); err != nil {
		return Project{}, fmt.Errorf("failed to decode compose file: %w", err)
	}

	return Project{
		XTopo: parsed.XTopo,
		URL:   source.URL(),
		Ref:   source.SHA,
	}, nil
}

func FetchProject(client GitHubClient, source GitHubSource) (Project, error) {
	yamlBytes, err := client.FetchFile(source, "compose.yaml")
	if err != nil {
		return Project{}, err
	}
	return NewProject(source, bytes.NewReader(yamlBytes))
}

func (t Project) SourceID() ProjectSourceID {
	return ProjectSourceID(t.URL)
}
