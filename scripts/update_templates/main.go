package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("⚠️ GITHUB_TOKEN is not set: you might get rate limited")
	}

	githubClient := NewGitHubClient(githubToken)

	var templates []Template
	for _, source := range ListGitHubSources(strings.NewReader(sourcesJSON)) {
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
		log.Fatalf("failed to find catalog output path: %v\n", err)
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

	return filepath.Join(repoRoot, "internal", "catalog", "data", "catalog.json"), nil
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
