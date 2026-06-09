package main

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

type Template struct {
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

func NewTemplate(source GitHubSource, composeFile io.Reader) (Template, error) {
	type composeDocument struct {
		XTopo XTopo `yaml:"x-topo"`
	}

	var parsed composeDocument
	decoder := yaml.NewDecoder(composeFile)
	if err := decoder.Decode(&parsed); err != nil {
		return Template{}, fmt.Errorf("failed to decode compose file: %w", err)
	}

	return Template{
		XTopo: parsed.XTopo,
		URL:   source.CloneURL(),
		Ref:   source.SHA,
	}, nil
}

func FetchTemplate(client GitHubClient, source GitHubSource) (Template, error) {
	yamlBytes, err := client.FetchFile(source, "compose.yaml")
	if err != nil {
		return Template{}, err
	}
	return NewTemplate(source, bytes.NewReader(yamlBytes))
}
