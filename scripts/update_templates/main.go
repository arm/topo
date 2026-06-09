package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

const outputJSONPath = "internal/catalog/data/catalog.json"

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

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "GITHUB_TOKEN is not set: create a personal access token and set the envvar")
		os.Exit(1)
	}

	client := &http.Client{}

	var templates []Template

	for _, source := range ListSources(strings.NewReader(sourcesJSON)) {
		composeBytes, err := fetchComposeFile(client, token, source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", source, err)
			continue
		}

		repoURL := fmt.Sprintf("https://github.com/%s.git", source.Repo)

		tmpl, err := BuildTemplate(repoURL, composeBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", source, err)
			continue
		}
		tmpl.Ref = source.SHA

		templates = append(templates, tmpl)
	}

	if err := WriteTemplates(outputJSONPath, templates); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s\n", outputJSONPath)
}
