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

type Catalog struct {
	Schema    string     `json:"$schema"`
	Templates []Template `json:"templates"`
}

func ReadTemplates() ([]Template, error) {
	path, err := catalogOutputPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck // Closing a read-only file cannot affect catalog generation.

	catalog, err := ReadCatalogFile(file)
	if err != nil {
		return nil, err
	}
	return catalog.Templates, nil
}

func ReadCatalogFile(r io.Reader) (Catalog, error) {
	var document Catalog
	if err := json.NewDecoder(r).Decode(&document); err != nil {
		return Catalog{}, err
	}
	return document, nil
}

func WriteTemplates(templates []Template) (string, error) {
	outputFile, outputFilePath, err := createCatalogOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create catalog output: %w", err)
	}
	writeErr := WriteTemplatesToCatalogFile(outputFile, templates)
	closeErr := outputFile.Close()
	if writeErr != nil {
		return "", fmt.Errorf("failed to write templates: %w", writeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("failed to close catalog output: %w", writeErr)
	}
	return outputFilePath, nil
}

func WriteTemplatesToCatalogFile(w io.Writer, templates []Template) error {
	document := Catalog{
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
