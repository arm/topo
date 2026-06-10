package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTemplates(t *testing.T) {
	t.Run("writes expected json", func(t *testing.T) {
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

		err := WriteTemplates(&output, input)
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
