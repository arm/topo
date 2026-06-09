package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/arm/topo/internal/template"
	"gopkg.in/yaml.v3"
)

type Template struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Features    []string       `json:"features"`
	Args        map[string]Arg `json:"args,omitempty"`
	URL         string         `json:"url"`
	Ref         string         `json:"ref"`
}

type Arg struct {
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Default     string         `json:"default,omitempty"`
	Example     string         `json:"example,omitempty"`
	Hints       map[string]any `json:"hints,omitempty"`
}

func NewTemplate(source GitHubSource, compose io.Reader) (Template, error) {
	content, err := io.ReadAll(compose)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read compose definition: %w", err)
	}

	tmpl, err := template.FromContent(bytes.NewReader(content))
	if err != nil {
		return Template{}, fmt.Errorf("failed to parse compose definition: %w", err)
	}

	metadata := tmpl.Metadata
	if metadata.Name == "" {
		return Template{}, fmt.Errorf("no valid x-topo name in compose definition")
	}

	args, err := parseArgs(content)
	if err != nil {
		return Template{}, fmt.Errorf("failed to parse args: %w", err)
	}

	return Template{
		Name:        metadata.Name,
		Description: metadata.Description,
		Features:    metadata.Features,
		Args:        args,
		URL:         source.URL(),
		Ref:         source.SHA,
	}, nil
}

func FetchTemplate(source GitHubSource, githubToken string) (Template, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, source.FileURL("compose.yaml"), nil)
	if err != nil {
		return Template{}, err
	}

	req.Header.Set("User-Agent", "topo-template-update")
	if githubToken != "" {
		req.Header.Set("Authorization", "token "+githubToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	// #nosec G704 -- request is validated, false positive warning
	resp, err := client.Do(req)
	if err != nil {
		return Template{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return Template{}, fmt.Errorf("compose.yaml not found (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return Template{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	yamlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read response body: %w", err)
	}
	return NewTemplate(source, bytes.NewReader(yamlBytes))
}

func parseArgs(compose []byte) (map[string]Arg, error) {
	type rawCompose struct {
		XTopo struct {
			Args map[string]Arg `yaml:"args"`
		} `yaml:"x-topo"`
	}

	var parsed rawCompose
	if err := yaml.Unmarshal(compose, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.XTopo.Args) == 0 {
		return nil, nil
	}

	return parsed.XTopo.Args, nil
}
