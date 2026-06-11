package main

import (
	"os"
	"path/filepath"
)

// TODO: Move files out of internal when we extract this to a separate repo
const relativeCatalogOutputPath = "internal/catalog/data/catalog.json"

const relativeSourcesPath = "scripts/update_templates/sources.json"

func catalogOutputPath() (string, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, filepath.FromSlash(relativeCatalogOutputPath)), nil
}

func createCatalogOutput() (*os.File, string, error) {
	path, err := catalogOutputPath()
	if err != nil {
		return nil, "", err
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, "", err
	}

	return file, path, nil
}

func openGitHubSources() (*os.File, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return nil, err
	}

	return os.Open(filepath.Join(repoRoot, filepath.FromSlash(relativeSourcesPath)))
}
