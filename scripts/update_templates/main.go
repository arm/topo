package main

import (
	"fmt"
	"os"
	"strings"
)

const outputJSONPath = "internal/catalog/data/catalog.json"

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")

	var templates []Template

	for _, source := range ListSources(strings.NewReader(sourcesJSON)) {
		template, err := FetchTemplate(source, githubToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", source, err)
			continue
		}

		templates = append(templates, template)
	}

	if err := WriteTemplates(outputJSONPath, templates); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s\n", outputJSONPath)
}
