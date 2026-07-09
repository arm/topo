package catalog

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed data/catalog.json
var catalogJSON []byte

//go:embed data/catalog.schema.json
var catalogSchemaJSON []byte

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

func ListBuiltinProjects() ([]Project, error) {
	return parseProjects(catalogJSON)
}

func ListProjectsFromURL(ctx context.Context, url string) ([]Project, error) {
	data, err := fetchProjectsJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	return parseProjects(data)
}

func parseProjects(b []byte) ([]Project, error) {
	if err := validateAgainstSchema(b); err != nil {
		return nil, fmt.Errorf("failed schema validation: %w", err)
	}

	var catalog catalogDocument
	if err := json.Unmarshal(b, &catalog); err != nil {
		return nil, fmt.Errorf("failed to unmarshal projects: %w", err)
	}

	return catalog.Projects, nil
}

func validateAgainstSchema(b []byte) error {
	const projectsSchemaURL = "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json"

	compiler := jsonschema.NewCompiler()
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(catalogSchemaJSON))
	if err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	if err := compiler.AddResource(projectsSchemaURL, schemaDoc); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}
	schema, err := compiler.Compile(projectsSchemaURL)
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	jsonDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("failed to unmarshal projects: %w", err)
	}
	return schema.Validate(jsonDoc)
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

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project: %w", err)
	}
	return data, nil
}

func httpGet(ctx context.Context, rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		parsedURL.String(),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- URL is explicitly provided by the CLI user and scheme-validated above.
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
