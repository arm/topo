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

	outputPath, err := catalogOutputPath()
	if err != nil {
		log.Fatalf("failed to calculate catalog output path: %v\n", err)
	}

	if err := WriteTemplates(outputPath, templates); err != nil {
		log.Printf("failed to write templates: %v\n", err)
		os.Exit(1)
	}
	log.Printf("written catalog to %s\n", outputPath)
}
