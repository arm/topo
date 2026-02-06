package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/arm-debug/topo-cli/configs"
)

// Template represents a service template definition (comes from embedded templates.json)
type Template struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	Subsystem string   `json:"subsystem"`
	Url       string   `json:"url"`
	Platform  string   `json:"platform,omitempty"`
	Ports     []string `json:"ports,omitempty"`
}

// ReadTemplates returns templates from embedded JSON.
func ReadTemplates() ([]Template, error) {
	var templates []Template
	dec := json.NewDecoder(bytes.NewReader(configs.TemplatesJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	for i := range templates { // ensure Ports non-nil
		if templates[i].Ports == nil {
			templates[i].Ports = []string{}
		}
	}
	return templates, nil
}

// ListTemplates emits templates JSON to stdout.
func ListTemplates() error {
	templates, err := ReadTemplates()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
