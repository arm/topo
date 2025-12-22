package main

import (
	"encoding/json"
	"os"
)

func WriteTemplates(path string, templates []Template) (err error) {
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

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(templates)
}
