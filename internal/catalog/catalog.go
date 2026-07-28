package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/fetch"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type catalogDocument struct {
	Schema   string    `json:"$schema,omitempty"`
	Projects []Project `json:"projects"`
}

type Project struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	URL         string   `json:"url"`
	Ref         string   `json:"ref"`
}

const (
	defaultURL        = "https://artifacts.tools.arm.com/devx-topo-project-catalog/" + version + "/catalog/"
	DefaultCatalogURL = defaultURL + "catalog.json"
	defaultSchemaURL  = defaultURL + "catalog.schema.json"
	version           = "v1"
)

func ListProjectsFromURL(ctx context.Context, url string) ([]Project, error) {
	data, err := fetchProjectsJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	return parseProjects(ctx, data)
}

func parseProjects(ctx context.Context, b []byte) ([]Project, error) {
	if err := validateAgainstSchema(ctx, b); err != nil {
		return nil, fmt.Errorf("failed schema validation: %w", err)
	}

	var catalog catalogDocument
	if err := json.Unmarshal(b, &catalog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal projects: %w", err)
	}

	return catalog.Projects, nil
}

func validateAgainstSchema(ctx context.Context, b []byte) error {
	compiler := jsonschema.NewCompiler()
	loader := schemaLoader{ctx: ctx}
	compiler.UseLoader(jsonschema.SchemeURLLoader{
		"http":  loader,
		"https": loader,
	})
	schema, err := compiler.Compile(defaultSchemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	jsonDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to unmarshal projects: %w", err)
	}
	return schema.Validate(jsonDoc)
}

type schemaLoader struct {
	ctx context.Context
}

func (l schemaLoader) Load(url string) (any, error) {
	data, err := fetch.Get(l.ctx, url)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(data))
}

func fetchProjectsJSON(ctx context.Context, url string) ([]byte, error) {
	const filePrefix = "file://"
	if path, found := strings.CutPrefix(url, filePrefix); found {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read projects: %w", err)
		}
		return data, nil
	}

	data, err := fetch.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project: %w", err)
	}
	return data, nil
}
