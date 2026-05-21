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
)

//go:embed data/templates.json
var TemplatesJSON []byte

type Repo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Features    []string `json:"features"`
	MinRAMKb    int64    `json:"min_ram_kb,omitempty"`
	URL         string   `json:"url"`
	Ref         string   `json:"ref"`
}

func ListBuiltinTemplates() ([]Repo, error) {
	return ParseTemplates(TemplatesJSON)
}

func ListTemplatesFromURL(ctx context.Context, url string) ([]Repo, error) {
	data, err := FetchTemplatesJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch templates: %w", err)
	}
	return ParseTemplates(data)
}

func ParseTemplates(b []byte) ([]Repo, error) {
	var templates []Repo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&templates); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}
	return templates, nil
}

func FetchTemplatesJSON(ctx context.Context, url string) ([]byte, error) {
	const filePrefix = "file://"
	if path, found := strings.CutPrefix(url, filePrefix); found {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read templates: %w", err)
		}
		return data, nil
	}

	data, err := httpGet(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
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

	client := http.Client{}
	resp, err := client.Do(req) // #nosec G704 -- URL is explicitly provided by the CLI user and scheme-validated above.
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
