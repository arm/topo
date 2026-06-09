package main

import (
	"log"
	"os"
	"strings"
)

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("⚠️ GITHUB_TOKEN is not set: you might get rate limited")
	}

	var templates []Template
	for _, source := range ListSources(strings.NewReader(sourcesJSON)) {
		template, err := FetchTemplate(source, githubToken)
		if err != nil {
			log.Printf("failed to fetch %s (%v)\n", source, err)
			continue
		}
		log.Printf("fetched %s\n", source)
		templates = append(templates, template)
	}

	const outputJSONPath = "internal/catalog/data/catalog.json"
	if err := WriteTemplates(outputJSONPath, templates); err != nil {
		log.Printf("failed to write templates: %v\n", err)
		os.Exit(1)
	}
	log.Printf("written catalog to %s\n", outputJSONPath)
}
