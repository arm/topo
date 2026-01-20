package main

import (
	"fmt"
	"net/http"
	"os"
)

const outputJSONPath = "internal/catalog/data/templates.json"

var repoList = []string{
	"Arm-Debug/topo-cortexa-welcome#main",
	"Arm-Debug/topo-kleidi-service#main",
	"Arm-Debug/STM32-Heteogenous-Communications-example#main",
	"Arm-Debug/topo-armv9-cpu-llm-chat#master",
	"Arm-Debug/topo-simd-visual-benchmark#master",
}

type Template struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	URL         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func main() {
	token := os.Getenv("GH_PAT")
	if token == "" {
		fmt.Fprintln(os.Stderr, "GH_PAT is not set: create a personal access token and set the envvar")
		os.Exit(1)
	}

	client := &http.Client{}

	var templates []Template

	seenIDs := make(map[string]struct{})

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

		if _, exists := seenIDs[tmpl.ID]; exists {
			fmt.Fprintf(os.Stderr, "duplicate template id %q from %s; skipping\n", tmpl.ID, spec)
			continue
		}

		seenIDs[tmpl.ID] = struct{}{}
		templates = append(templates, tmpl)
	}

	if err := WriteTemplates(outputJSONPath, templates); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s\n", outputJSONPath)
}
