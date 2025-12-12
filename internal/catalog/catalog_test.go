package catalog_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/arm-debug/topo-cli/internal/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTemplateRepo(t *testing.T) {
	t.Run("when template exists it is found", func(t *testing.T) {
		template, err := catalog.GetTemplateRepo("kleidi-llm")

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{Id: "kleidi-llm", Url: "git@github.com:Arm-Debug/topo-kleidi-service.git"}, template)
	})

	t.Run("when template does not exist, it errors", func(t *testing.T) {
		_, err := catalog.GetTemplateRepo("nonexistent-template")

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent-template" not found`)
	})
}

func TestPrintTemplateRepos(t *testing.T) {
	t.Run("prints templates as JSON", func(t *testing.T) {
		var buf bytes.Buffer

		err := catalog.PrintTemplateRepos(&buf)

		require.NoError(t, err)
		var templates []catalog.Repo
		require.NoError(t, json.Unmarshal(buf.Bytes(), &templates))
		assert.NotEmpty(t, templates)
	})
}

func TestListRepos(t *testing.T) {
	t.Run("parses valid JSON successfully", func(t *testing.T) {
		jsonData := []byte(`[
			{"id": "test-repo", "url": "https://example.com/repo.git"},
			{"id": "another-repo", "url": "https://example.com/another.git", "ref": "v1.0.0"}
		]`)

		templates, err := catalog.ListRepos(jsonData)

		require.NoError(t, err)
		assert.Len(t, templates, 2)
		assert.Equal(t, catalog.Repo{Id: "test-repo", Url: "https://example.com/repo.git"}, templates[0])
		assert.Equal(t, catalog.Repo{Id: "another-repo", Url: "https://example.com/another.git", Ref: "v1.0.0"}, templates[1])
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		jsonData := []byte(`[{"id": "test", invalid}]`)

		_, err := catalog.ListRepos(jsonData)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})

	t.Run("returns error for unknown fields", func(t *testing.T) {
		jsonData := []byte(`[{"id": "test", "url": "https://example.com", "unknown_field": "value"}]`)

		_, err := catalog.ListRepos(jsonData)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})
}

func TestGetRepo(t *testing.T) {
	validJSON := []byte(`[
		{"id": "repo1", "url": "https://example.com/repo1.git"},
		{"id": "repo2", "url": "https://example.com/repo2.git", "ref": "main"}
	]`)

	t.Run("finds existing repo by id", func(t *testing.T) {
		repo, err := catalog.GetRepo("repo1", validJSON)

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{Id: "repo1", Url: "https://example.com/repo1.git"}, repo)
	})

	t.Run("finds repo with ref", func(t *testing.T) {
		repo, err := catalog.GetRepo("repo2", validJSON)

		require.NoError(t, err)
		assert.Equal(t, &catalog.Repo{Id: "repo2", Url: "https://example.com/repo2.git", Ref: "main"}, repo)
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
