package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTemplates(t *testing.T) {
	t.Run("reads templates from catalog file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")
		err := os.WriteFile(path, []byte(`
{
	"$schema": "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json",
	"templates": [
		{
			"name": "death-star-trench-run",
			"description": "Use the Force to benchmark impossible shots",
			"features": ["X-wing", "Astromech", "Proton torpedoes"],
			"url": "ssh://death-star.example",
			"ref": "rebellion"
		}
	]
}
`), 0o600)
		require.NoError(t, err)

		got, err := ReadTemplates(path)

		require.NoError(t, err)
		want := []Template{
			{
				XTopo: XTopo{
					Name:        "death-star-trench-run",
					Description: "Use the Force to benchmark impossible shots",
					Features:    []string{"X-wing", "Astromech", "Proton torpedoes"},
				},
				URL: "ssh://death-star.example",
				Ref: "rebellion",
			},
		}
		assert.Equal(t, want, got)
	})
}

func TestWriteCatalog(t *testing.T) {
	t.Run("writes catalog document to file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")
		want := Catalog{
			Schema: "https://raw.githubusercontent.com/arm/topo/main/internal/catalog/data/catalog.schema.json",
			Templates: []Template{
				{
					XTopo: XTopo{
						Name:        "death-star-trench-run",
						Description: "Use the Force to benchmark impossible shots",
						Features:    []string{"X-wing", "Astromech", "Proton torpedoes"},
					},
					URL: "ssh://death-star.example",
					Ref: "rebellion",
				},
			},
		}

		err := WriteCatalog(path, want)

		require.NoError(t, err)
		gotBytes, err := os.ReadFile(path)
		require.NoError(t, err)
		var got Catalog
		err = json.Unmarshal(gotBytes, &got)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
