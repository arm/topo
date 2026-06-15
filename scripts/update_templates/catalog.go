package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	relativeCatalogOutputPath = "internal/catalog/data/catalog.json"
)

type Catalog struct {
	Schema    string     `json:"$schema"`
	Templates []Template `json:"templates"`
}

func ReadTemplates(path string) ([]Template, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck // Closing a read-only file cannot affect catalog generation.

	var document Catalog
	if err := json.NewDecoder(file).Decode(&document); err != nil {
		return nil, err
	}
	return document.Templates, nil
}

func WriteTemplates(path string, templates []Template, validator CatalogSchema) error {
	document := Catalog{
		Schema:    validator.SchemaURL(),
		Templates: templates,
	}
	if err := validator.ValidateCatalog(document); err != nil {
		return fmt.Errorf("invalid catalog document: %w", err)
	}

	outputFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create catalog output: %w", err)
	}
	enc := json.NewEncoder(outputFile)
	enc.SetIndent("", "  ")
	writeErr := enc.Encode(document)
	closeErr := outputFile.Close()
	if writeErr != nil {
		return fmt.Errorf("failed to write templates: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close catalog output: %w", closeErr)
	}
	return nil
}

func CatalogFilePath() (string, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, filepath.FromSlash(relativeCatalogOutputPath)), nil
}
