package catalog

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func ListTemplatesFromURL(url string) ([]Repo, error) {
	data, err := FetchTemplatesJSON(url)
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

func FetchTemplatesJSON(url string) ([]byte, error) {
	const filePrefix = "file://"
	if path, found := strings.CutPrefix(url, filePrefix); found {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read templates: %w", err)
		}
		return data, nil
	}

	data, err := httpGet(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}
	return data, nil
}

func httpGet(url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		url,
		nil,
	)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}
