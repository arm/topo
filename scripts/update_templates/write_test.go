package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTemplates(t *testing.T) {
	t.Run("writes expected json", func(t *testing.T) {
		writeTo := filepath.Join(t.TempDir(), "templates.json")
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

		err := WriteTemplates(writeTo, input)
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
		assert.JSONEq(t, want, requireReadFile(t, writeTo))
	})
}

func requireReadFile(t testing.TB, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
