package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	relativeCatalogOutputPath = "internal/catalog/data/catalog.json"
	catalogSchemaURL          = "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json"
)

type CatalogDocument struct {
	Schema    string     `json:"$schema"`
	Templates []Template `json:"templates"`
}

func WriteCatalogFile(templates []Template) (string, error) {
	outputFile, outputFilePath, err := createCatalogOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create catalog output: %w", err)
	}
	writeErr := WriteCatalog(outputFile, templates)
	closeErr := outputFile.Close()
	if writeErr != nil {
		return "", fmt.Errorf("failed to write templates: %w", writeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("failed to close catalog output: %w", writeErr)
	}
	return outputFilePath, nil
}

func WriteCatalog(w io.Writer, templates []Template) error {
	document := CatalogDocument{
		Schema:    catalogSchemaURL,
		Templates: templates,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(document)
}

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
		return nil, path, err
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, path, err
	}

	return file, path, nil
}
