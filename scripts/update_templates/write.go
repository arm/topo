package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const catalogSchemaURL = "https://topo.arm.com/schemas/templates/1/schema.json"

type Catalog struct {
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

	catalog := Catalog{
		Schema:    catalogSchemaURL,
		Templates: templates,
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(catalog)
}
