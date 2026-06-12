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

	sources, err := ListGitHubSources()
	if err != nil {
		log.Fatalf("failed to list sources: %v\n", err)
	}

	catalog, err := ReadCatalogFile()
	if err != nil {
		log.Fatalf("failed to read catalog file: %v\n", err)
	}

	plan := PlanUpdate(sources, catalog.Templates)
	log.Printf(
		"template update plan: %d add, %d update, %d remove, %d unchanged",
		len(plan.ToAdd),
		len(plan.ToUpdate),
		len(plan.ToRemove),
		len(plan.Unchanged),
	)

	templates := append([]Template{}, plan.Unchanged...)
	for _, source := range append(plan.ToAdd, plan.ToUpdate...) {
		template, err := FetchTemplate(githubClient, source)
		if err != nil {
			log.Fatalf("failed to fetch %s: %v\n", source, err)
		}
		log.Printf("fetched %s\n", source)
		templates = append(templates, template)
	}

	filePath, err := WriteCatalogFile(templates)
	if err != nil {
		log.Fatalf("failed to write catalog file: %v\n", err)
	}
	log.Printf("written catalog to %s\n", filePath)
}
