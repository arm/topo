package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const catalogSchemaURL = "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json"

type CatalogDocument struct {
	Schema    string     `json:"$schema"`
	Templates []Template `json:"templates"`
}

func WriteTemplates(path string, templates []Template) (err error) {
	path = filepath.Clean(path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	document := CatalogDocument{
		Schema:    catalogSchemaURL,
		Templates: templates,
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(document)
}
