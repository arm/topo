package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

const outputJSONPath = "internal/catalog/data/catalog.json"

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

		tmpl, err := BuildTemplate(source, composeBytes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping %s: %v\n", source, err)
			continue
		}

		templates = append(templates, tmpl)
	}

	if err := WriteTemplates(outputJSONPath, templates); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write templates: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s\n", outputJSONPath)
}
