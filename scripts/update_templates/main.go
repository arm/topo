package main

import (
	"log"
	"os"
	"strings"
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
	log.Printf("update plan:\n%s", indent(plan.String()))
	if !plan.HasChanges() {
		log.Println("catalog already up to date")
		return
	}

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

func indent(text string) string {
	return "  " + strings.ReplaceAll(text, "\n", "\n  ")
}
