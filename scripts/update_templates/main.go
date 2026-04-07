package main

import (
	"fmt"
	"net/http"
	"os"
)

const outputJSONPath = "internal/catalog/data/templates.json"

var repoList = []string{
	"Arm-Examples/topo-welcome#main",
	"Arm-Examples/topo-lightbulb-moment#main",
	"Arm-Examples/topo-cpu-ai-chat#main",
	"Arm-Examples/topo-simd-visual-benchmark#main",
}

type Template struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	URL         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "GITHUB_TOKEN is not set: create a personal access token and set the envvar")
		os.Exit(1)
	}

	client := &http.Client{}

	var templates []Template

	seenNames := make(map[string]struct{})

	for _, spec := range repoList {
		repo, ref := parseRepoSpec(spec)

		composeBytes, err := fetchComposeFile(client, token, spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", spec, err)
			continue
		}

		repoURL := fmt.Sprintf("git@github.com:%s.git", repo)

		tmpl, err := BuildTemplate(repoURL, composeBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", spec, err)
			continue
		}
		tmpl.Ref = ref

		if _, exists := seenNames[tmpl.Name]; exists {
			panic(fmt.Sprintf("duplicate template name %q from %s; skipping\n", tmpl.Name, spec))
		}

		seenNames[tmpl.Name] = struct{}{}
		templates = append(templates, tmpl)
	}

	if err := WriteTemplates(outputJSONPath, templates); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s\n", outputJSONPath)
}
