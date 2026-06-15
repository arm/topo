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

func ReadTemplates(path string) ([]Template, error) {
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
	writeErr := WriteCatalogFile(outputFile, document)
	closeErr := outputFile.Close()
	if writeErr != nil {
		return fmt.Errorf("failed to write templates: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close catalog output: %w", closeErr)
	}
	return nil
}

func WriteCatalogFile(w io.Writer, document Catalog) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(document)
}

func CatalogFilePath() (string, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, filepath.FromSlash(relativeCatalogOutputPath)), nil
}
