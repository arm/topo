package main

import (
	"encoding/json"
	"io"
)

const catalogSchemaURL = "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json"

type CatalogDocument struct {
	Schema    string     `json:"$schema"`
	Templates []Template `json:"templates"`
}

func WriteTemplates(w io.Writer, templates []Template) error {
	document := CatalogDocument{
		Schema:    catalogSchemaURL,
		Templates: templates,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(document)
}
