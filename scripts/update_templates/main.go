package main

import (
	"log"
	"os"
)

func main() {
	sources, err := ListGitHubSources()
	if err != nil {
		log.Fatalf("failed to list sources: %v\n", err)
	}

	currentTemplates, err := ReadTemplates()
	if err != nil {
		log.Fatalf("failed to read catalog file: %v\n", err)
	}

	plan := PlanUpdate(sources, currentTemplates)
	if !plan.HasChanges() {
		log.Println("catalog already up to date")
		return
	}
	log.Printf(
		"template update plan:\n  🆕 %d to add\n  🔄 %d to update\n  🗑️ %d to remove\n  ☑️ %d unchanged",
		len(plan.ToAdd),
		len(plan.ToUpdate),
		len(plan.ToRemove),
		len(plan.Unchanged),
	)

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("⚠️ GITHUB_TOKEN is not set: you might get rate limited")
	}
	githubClient := NewGitHubClient(githubToken)

	templates := append([]Template{}, plan.Unchanged...)
	for _, source := range append(plan.ToAdd, plan.ToUpdate...) {
		template, err := FetchTemplate(githubClient, source)
		if err != nil {
			log.Fatalf("failed to fetch %s: %v\n", source, err)
		}
		log.Printf("fetched %s\n", source)
		templates = append(templates, template)
	}
	templates = TemplatesInSourceOrder(sources, templates)

	filePath, err := WriteTemplates(templates)
	if err != nil {
		log.Fatalf("failed to write catalog file: %v\n", err)
	}
	log.Printf("written catalog to %s\n", filePath)
}
