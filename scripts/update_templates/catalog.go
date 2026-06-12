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

func ReadCatalogFile() (Catalog, error) {
	path, err := catalogOutputPath()
	if err != nil {
		return Catalog{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		return Catalog{}, err
	}
	defer file.Close() //nolint:errcheck // Closing a read-only file cannot affect catalog generation.

	return ReadCatalog(file)
}

func ReadCatalog(r io.Reader) (Catalog, error) {
	var document Catalog
	if err := json.NewDecoder(r).Decode(&document); err != nil {
		return Catalog{}, err
	}
	return document, nil
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
