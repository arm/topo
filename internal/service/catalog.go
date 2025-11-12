package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/arm-debug/topo-cli/configs"
)

type TemplateRepo struct {
	Id  string `json:"id"`
	Url string `json:"url"`
	Ref string `json:"ref,omitempty"`
}

func ListTemplateRepos() ([]TemplateRepo, error) {
	var templates []TemplateRepo
	dec := json.NewDecoder(bytes.NewReader(configs.ServiceTemplatesJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func GetTemplateRepo(id string) (*TemplateRepo, error) {
	templates, err := ListTemplateRepos()
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

func PrintTemplateRepos(w io.Writer) error {
	templates, err := ListTemplateRepos()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}
	fmt.Fprintf(w, "%s\n", data)
	return nil
}
