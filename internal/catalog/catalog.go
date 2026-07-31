package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/fetch"
)

type Project = ProjectElement

var (
	defaultURL        = "https://artifacts.tools.arm.com/devx-topo-project-catalog/" + majorVersion(CatalogSchemaVersion) + "/catalog/"
	DefaultCatalogURL = defaultURL + "catalog.json"
)

func majorVersion(version string) string {
	major, _, _ := strings.Cut(version, ".")
	return major
}

func ListProjectsFromURL(ctx context.Context, url string) ([]Project, error) {
	data, err := fetchProjectsJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	return parseProjects(data)
}

func parseProjects(b []byte) ([]Project, error) {
	catalogVersion, versionErr := unmarshalCatalogVersion(b)
	catalog, err := UnmarshalCatalogDocument(b)
	if err != nil {
		if versionErr == nil && catalogVersion != "" && catalogVersion != CatalogSchemaVersion {
			return nil, fmt.Errorf(
				"failed to unmarshal catalog: catalog version %q differs from generated schema version %q: %w",
				catalogVersion,
				CatalogSchemaVersion,
				err,
			)
		}
		return nil, fmt.Errorf("failed to unmarshal catalog: %w", err)
	}
	return catalog.Projects, nil
}

func unmarshalCatalogVersion(b []byte) (string, error) {
	var header struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(b, &header); err != nil {
		return "", err
	}
	return header.Version, nil
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
