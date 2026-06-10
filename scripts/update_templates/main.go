package main

import (
	"log"
	"os"
)

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("⚠️ GITHUB_TOKEN is not set: you might get rate limited")
	}

	githubClient := NewGitHubClient(githubToken)

	sourcesFile, err := openGitHubSources()
	if err != nil {
		log.Fatalf("failed to open GitHub sources: %v\n", err)
	}
	defer sourcesFile.Close()

	var templates []Template
	for _, source := range ListGitHubSources(sourcesFile) {
		template, err := FetchTemplate(githubClient, source)
		if err != nil {
			log.Printf("failed to fetch %s (%v)\n", source, err)
			continue
		}
		log.Printf("fetched %s\n", source)
		templates = append(templates, template)
	}

	outputFile, outputPath, err := createCatalogOutput()
	if err != nil {
		log.Fatalf("failed to create catalog output: %v\n", err)
	}

	writeErr := WriteTemplates(outputFile, templates)
	closeErr := outputFile.Close()
	if writeErr != nil {
		log.Printf("failed to write templates: %v\n", writeErr)
		os.Exit(1)
	}
	if closeErr != nil {
		log.Printf("failed to close catalog output: %v\n", closeErr)
		os.Exit(1)
	}
	log.Printf("written catalog to %s\n", outputPath)
}
