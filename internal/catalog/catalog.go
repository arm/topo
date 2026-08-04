package catalog

//go:generate go run ../../scripts/generate_catalog_types v2.0.0

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
	catalogVersionMajor := majorVersion(catalogVersion)
	SchemaVersionMajor := majorVersion(CatalogSchemaVersion)
	if catalogVersionMajor != SchemaVersionMajor {
		return nil, fmt.Errorf(
			"failed to parse catalog: requested catalog version %q is incompatible with supported schema version %q: %w",
			catalogVersion,
			CatalogSchemaVersion,
			versionErr,
		)
	}
	catalog, err := UnmarshalCatalogDocument(b)
	if err != nil {
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
