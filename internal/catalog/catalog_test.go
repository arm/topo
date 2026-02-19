package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTemplateRepo(t *testing.T) {
	t.Run("when template exists it is found", func(t *testing.T) {
		template, err := catalog.GetTemplateRepo("Lightbulb-moment")

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{
			Id:          "Lightbulb-moment",
			Description: "Reads a switch over GPIO pins on an M class cpu, reports switch state over Remoteproc Message, then a web application on the A class reads this and displays a lightbulb in either the on or off state. The lightbulb state is described by an LLM in any user-specified style.",
			Features:    []string{"SVE", "NEON"},
			Url:         "git@github.com:Arm-Examples/topo-template-lightbulb-moment.git",
			Ref:         "main",
		}, template)
	})

	t.Run("when template does not exist, it errors", func(t *testing.T) {
		_, err := catalog.GetTemplateRepo("nonexistent-template")

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent-template" not found`)
	})
}

func TestFilterTemplateRepos(t *testing.T) {
	t.Run("correctly filters for single search feature", func(t *testing.T) {
		flags := catalog.TemplateFilters{
			Features: []string{"walnut"},
		}

		collection := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Features:    []string{"walnut"},
				Url:         "url.git",
				Ref:         "main",
			},
			{
				Id:          "name-of-other-project",
				Description: "blah blah blah",
				Features:    []string{"almond"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		got, err := catalog.FilterTemplateRepos(flags, collection)
		require.NoError(t, err)

		want := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Features:    []string{"walnut"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		assert.Equal(t, want, got)
	})

	t.Run("is case insensitive", func(t *testing.T) {
		flags := catalog.TemplateFilters{
			Features: []string{"walnut"},
		}

		collection := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Features:    []string{"WALNUT"},
				Url:         "url.git",
				Ref:         "main",
			},
			{
				Id:          "name-of-other-project",
				Description: "blah blah blah",
				Features:    []string{"almond"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		got, err := catalog.FilterTemplateRepos(flags, collection)
		require.NoError(t, err)

		want := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Features:    []string{"WALNUT"},
				Url:         "url.git",
				Ref:         "main",
			},
		}

		assert.Equal(t, want, got)
	})
}

func TestListRepos(t *testing.T) {
	t.Run("parses valid JSON successfully", func(t *testing.T) {
		jsonData := []byte(`[
			{
				"id": "test-repo",
				"description": "A test template",
				"features": ["feat1", "feat2"],
				"url": "https://example.com/repo.git",
				"ref": "main"
			},
			{
				"id": "another-repo",
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
			Id:          "test-repo",
			Description: "A test template",
			Features:    []string{"feat1", "feat2"},
			Url:         "https://example.com/repo.git",
			Ref:         "main",
		}, templates[0])
		assert.Equal(t, catalog.Repo{
			Id:          "another-repo",
			Description: "Another template",
			Features:    nil,
			Url:         "https://example.com/another.git",
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
				"id": "test",
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
			"id": "repo1",
			"description": "first",
			"features": ["feat"],
			"url": "https://example.com/repo1.git"
		},
		{
			"id": "repo2",
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
			Id:          "repo1",
			Description: "first",
			Features:    []string{"feat"},
			Url:         "https://example.com/repo1.git",
		}, repo)
	})

	t.Run("finds repo with ref", func(t *testing.T) {
		repo, err := catalog.GetRepo("repo2", validJSON)

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{
			Id:          "repo2",
			Description: "second",
			Features:    nil,
			Url:         "https://example.com/repo2.git",
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
