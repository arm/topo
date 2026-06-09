package main

import (
	"log"
	"os"
	"path/filepath"
)

// TODO: Move files out of internal when we extract this to a separate repo
const relativeCatalogOutputPath = "internal/catalog/data/catalog.json"

const relativeSourcesPath = "scripts/update_templates/sources.json"

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

func catalogOutputPath() (string, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, filepath.FromSlash(relativeCatalogOutputPath)), nil
}

func openGitHubSources() (*os.File, error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return nil, err
	}

	return os.Open(filepath.Join(repoRoot, filepath.FromSlash(relativeSourcesPath)))
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
