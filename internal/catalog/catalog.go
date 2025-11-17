package catalog

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

func GetExampleProjectRepo(id string) (*Repo, error) {
	return GetRepo(id, configs.ExampleProjectsJSON)
}

func PrintExampleProjectRepos(w io.Writer) error {
	return printRepos(w, configs.ExampleProjectsJSON)
}

func GetServiceTemplateRepo(id string) (*Repo, error) {
	return GetRepo(id, configs.ServiceTemplatesJSON)
}

func PrintServiceTemplateRepos(w io.Writer) error {
	return printRepos(w, configs.ServiceTemplatesJSON)
}

func ListRepos(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetRepo(id string, b []byte) (*Repo, error) {
	repos, err := ListRepos(b)
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

func printRepos(w io.Writer, b []byte) error {
	repos, err := ListRepos(b)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}
