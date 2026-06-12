package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCatalog(t *testing.T) {
	t.Run("reads templates from catalog input", func(t *testing.T) {
		input := bytes.NewBufferString(`
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
`)

		got, err := ReadCatalog(input)

		require.NoError(t, err)
		want := Catalog{
			Schema: catalogSchemaURL,
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
		assert.Equal(t, want, got)
	})
}

func TestWriteCatalog(t *testing.T) {
	t.Run("writes templates to catalog file", func(t *testing.T) {
		var output bytes.Buffer
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

		err := WriteCatalog(&output, input)
		require.NoError(t, err)

		want := `
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
`
		assert.JSONEq(t, want, output.String())
	})
}
