package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTemplateRepo(t *testing.T) {
	t.Run("when template exists it is found", func(t *testing.T) {
		template, err := catalog.GetTemplateRepo("topo-lightbulb-moment")

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{
			Name:        "topo-lightbulb-moment",
			Description: "Reads a switch over GPIO pins on an M class cpu, reports switch state over Remoteproc Message, then a web application on the A class reads this and displays a lightbulb in either the on or off state. The lightbulb state is described by an LLM in any user-specified style.",
			Features:    []string{"remoteproc-runtime"},
			URL:         "git@github.com:Arm-Examples/topo-lightbulb-moment.git",
			Ref:         "main",
		}, template)
	})
	t.Run("when template does not exist, it errors", func(t *testing.T) {
		_, err := catalog.GetTemplateRepo("nonexistent-template")

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent-template" not found`)
	})
}

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

func TestGetRepo(t *testing.T) {
	validJSON := []byte(`[
		{
			"name": "repo1",
			"description": "first",
			"features": ["feat"],
			"url": "https://example.com/repo1.git"
		},
		{
			"name": "repo2",
			"description": "second",
			"features": null,
			"url": "https://example.com/repo2.git",
			"ref": "main"
		}
	]`)

	t.Run("finds existing repo by id", func(t *testing.T) {
		repo, err := catalog.GetRepo("repo1", validJSON)

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{
			Name:        "repo1",
			Description: "first",
			Features:    []string{"feat"},
			URL:         "https://example.com/repo1.git",
		}, repo)
	})

	t.Run("finds repo with ref", func(t *testing.T) {
		repo, err := catalog.GetRepo("repo2", validJSON)

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{
			Name:        "repo2",
			Description: "second",
			Features:    nil,
			URL:         "https://example.com/repo2.git",
			Ref:         "main",
		}, repo)
	})

	t.Run("returns error when repo not found", func(t *testing.T) {
		_, err := catalog.GetRepo("nonexistent", validJSON)

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent" not found`)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := catalog.GetRepo("any-id", []byte(`invalid json`))

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})
}
