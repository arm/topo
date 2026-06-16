package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	relativeCatalogSchemaPath = "internal/catalog/data/catalog.schema.json"
	catalogSchemaURL          = "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json"
)

type CatalogSchema struct {
	schemaURL string
	schema    *jsonschema.Schema
}

func NewCatalogSchema(path string) (CatalogSchema, error) {
	schemaJSON, err := os.ReadFile(path)
	if err != nil {
		return CatalogSchema{}, fmt.Errorf("failed to read catalog schema: %w", err)
	}
	return NewCatalogSchemaFromBytes(schemaJSON)
}

func NewCatalogSchemaFromBytes(schemaJSON []byte) (CatalogSchema, error) {
	compiler := jsonschema.NewCompiler()
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return CatalogSchema{}, fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	if err := compiler.AddResource(catalogSchemaURL, schemaDoc); err != nil {
		return CatalogSchema{}, fmt.Errorf("failed to add schema resource: %w", err)
	}
	schema, err := compiler.Compile(catalogSchemaURL)
	if err != nil {
		return CatalogSchema{}, fmt.Errorf("failed to compile schema: %w", err)
	}

	return CatalogSchema{
		schemaURL: catalogSchemaURL,
		schema:    schema,
	}, nil
}

func (v CatalogSchema) SchemaURL() string {
	return v.schemaURL
}

func (v CatalogSchema) ValidateTemplate(template Template) error {
	document := Catalog{
		Schema:    v.SchemaURL(),
		Templates: []Template{template},
	}
	if err := v.ValidateCatalog(document); err != nil {
		return fmt.Errorf("invalid template document: %w", err)
	}
	return nil
}

func (v CatalogSchema) ValidateCatalog(document Catalog) error {
	jsonBytes, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	jsonDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to unmarshal catalog: %w", err)
	}
	if err := v.schema.Validate(jsonDoc); err != nil {
		return fmt.Errorf("failed schema validation: %w", err)
	}
	return nil
}

func CatalogSchemaFilePath() (string, error) {
	repoRoot, err := findModuleRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(repoRoot, filepath.FromSlash(relativeCatalogSchemaPath)), nil
}
