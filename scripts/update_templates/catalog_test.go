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

func TestWriteTemplates(t *testing.T) {
	t.Run("writes valid templates to catalog file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")
		validator := newTestCatalogSchema(t)
		input := []Template{
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

		err := WriteTemplates(path, input, validator)

		require.NoError(t, err)
		gotBytes, err := os.ReadFile(path)
		require.NoError(t, err)
		var got Catalog
		err = json.Unmarshal(gotBytes, &got)
		require.NoError(t, err)
		want := Catalog{
			Schema:    validator.SchemaURL(),
			Templates: input,
		}
		assert.Equal(t, want, got)
	})

	t.Run("does not overwrite catalog file when templates are invalid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "catalog.json")
		want := []byte("existing catalog")
		err := os.WriteFile(path, want, 0o600)
		require.NoError(t, err)
		validator := newTestCatalogSchema(t)
		input := []Template{
			{
				XTopo: XTopo{
					Description: "Missing a name",
				},
				URL: "https://github.com/Arm-Examples/topo-welcome.git",
				Ref: "main",
			},
		}

		err = WriteTemplates(path, input, validator)

		assert.Error(t, err)
		got, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})
}
