package catalog

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/arm/topo/internal/fetch"
)

type Project = ProjectElement

const (
	defaultURL        = "https://artifacts.tools.arm.com/devx-topo-project-catalog/" + CatalogMajorVersion + "/catalog/"
	DefaultCatalogURL = defaultURL + "catalog.json"
)

func ListProjectsFromURL(ctx context.Context, url string) ([]Project, error) {
	data, err := fetchProjectsJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch projects: %w", err)
	}
	return parseProjects(ctx, data)
}

func parseProjects(ctx context.Context, b []byte) ([]Project, error) {
	catalog, err := UnmarshalCatalogDocument(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal catalog: %w", err)
	}
	return catalog.Projects, nil
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
