package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRepos(t *testing.T) {
	t.Run("parses valid JSON successfully", func(t *testing.T) {
		jsonData := []byte(`[
			{
				"name": "test-repo",
				"description": "A test template",
				"features": ["feat1", "feat2"],
				"url": "https://example.com/repo.git",
				"ref": "main"
			},
			{
				"name": "another-repo",
				"description": "Another template",
				"features": null,
				"url": "https://example.com/another.git",
				"ref": "v1.0.0"
			}
		]`)

		templates, err := catalog.ParseRepos(jsonData)

		require.NoError(t, err)
		assert.Len(t, templates, 2)
		assert.Equal(t, catalog.Repo{
			Name:        "test-repo",
			Description: "A test template",
			Features:    []string{"feat1", "feat2"},
			URL:         "https://example.com/repo.git",
			Ref:         "main",
		}, templates[0])
		assert.Equal(t, catalog.Repo{
			Name:        "another-repo",
			Description: "Another template",
			Features:    nil,
			URL:         "https://example.com/another.git",
			Ref:         "v1.0.0",
		}, templates[1])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		jsonData := []byte(`[{"id": "test", invalid}]`)

		_, err := catalog.ParseRepos(jsonData)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})

	t.Run("returns error for unknown fields", func(t *testing.T) {
		jsonData := []byte(`[
			{
				"name": "test",
				"description": "desc",
				"features": [],
				"url": "https://example.com",
				"unknown_field": "value"
			}
		]`)

		_, err := catalog.ParseRepos(jsonData)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})
}
