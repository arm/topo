package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/arm-debug/topo-cli/configs"
)

type Repo struct {
	Id  string `json:"id"`
	Url string `json:"url"`
	Ref string `json:"ref,omitempty"`
}

func ListTemplateRepos(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetRepo(id string, b []byte) (*Repo, error) {
	templates, err := ListTemplateRepos(b)
	if err != nil {
		return nil, err
	}
	for i := range templates {
		if templates[i].Id == id {
			return &templates[i], nil
		}
	}
	return nil, fmt.Errorf("Service Template with id %q not found", id)
}

func GetTemplateRepo(id string) (*Repo, error) {
	return GetRepo(id, configs.ServiceTemplatesJSON)
}

func GetExampleProjectRepo(id string) (*Repo, error) {
	return GetRepo(id, configs.ExampleProjectsJSON)
}

func getRepos(b []byte) ([]byte, error) {
	templates, err := ListTemplateRepos(b)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal templates: %w", err)
	}
	return data, nil
}

func PrintExampleProjectRepos(w io.Writer) error {
	data, err := getRepos(configs.ExampleProjectsJSON)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}

func PrintTemplateRepos(w io.Writer) error {
	data, err := getRepos(configs.ServiceTemplatesJSON)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}
