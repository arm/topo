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

	githubClient := NewGitHubClient(githubToken)

	sourcesFilePath, err := GithubSourcesFilePath()
	if err != nil {
		log.Fatalf("failed to find sources file: %v\n", err)
	}

	sources, err := ListGithubSources(sourcesFilePath)
	if err != nil {
		log.Fatalf("failed to list sources: %v\n", err)
	}

	catalogFilePath, err := CatalogFilePath()
	if err != nil {
		log.Fatalf("failed to find catalog file: %v\n", err)
	}

	currentTemplates, err := ReadTemplates(catalogFilePath)
	if err != nil {
		log.Fatalf("failed to read catalog file: %v\n", err)
	}

	plan := PlanUpdate(sources, currentTemplates)
	log.Printf("update plan:\n%s", indent(plan.String()))
	if !plan.HasChanges() {
		log.Println("catalog already up to date")
		return
	}

	catalogSchemaFilePath, err := CatalogSchemaFilePath()
	if err != nil {
		log.Fatalf("failed to find catalog schema file: %v\n", err)
	}

	validator, err := NewCatalogSchema(catalogSchemaFilePath)
	if err != nil {
		log.Fatalf("failed to create schema validator: %v\n", err)
	}

	templates := append([]Template{}, plan.Unchanged...)
	for _, source := range append(plan.ToAdd, plan.ToUpdate...) {
		template, err := FetchTemplate(githubClient, source)
		if err != nil {
			log.Fatalf("failed to fetch %s: %v\n", source, err)
		}
		if err := validator.ValidateTemplate(template); err != nil {
			log.Fatalf("invalid template %s: %v\n", source, err)
		}
		log.Printf("fetched %s\n", source)
		templates = append(templates, template)
	}
	templates = TemplatesInSourceOrder(sources, templates)

	document := Catalog{
		Schema:    validator.SchemaURL(),
		Templates: templates,
	}
	if err := validator.ValidateCatalog(document); err != nil {
		log.Fatalf("invalid catalog file: %v\n", err)
	}
	if err := WriteCatalog(catalogFilePath, document); err != nil {
		log.Fatalf("failed to write catalog file: %v\n", err)
	}
	log.Printf("written catalog to %s\n", catalogFilePath)
}

func indent(text string) string {
	return "  " + strings.ReplaceAll(text, "\n", "\n  ")
}
