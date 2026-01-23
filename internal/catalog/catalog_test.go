package catalog_test

import (
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
		assert.Equal(t, &catalog.Repo{
			Id:          "kleidi-llm",
			Description: "Run an LLM locally using KleidiAI optimised inference on Arm CPU\n",
			Features:    []string{"SME", "NEON"},
			Url:         "git@github.com:Arm-Debug/topo-kleidi-service.git",
			Ref:         "main",
		}, template)
	})

	t.Run("when template does not exist, it errors", func(t *testing.T) {
		_, err := catalog.GetTemplateRepo("nonexistent-template")

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent-template" not found`)
	})
}

func TestPrintTemplateRepos(t *testing.T) {
	t.Run("prints multiple items correctly", func(t *testing.T) {
		dummyJSON := []byte(`[
  {
    "id": "name-of-project",
    "description": "blah blah blah",
    "features": null,
    "url": "url.git",
    "ref": "main"
  },
  {
    "id": "name-of-other-project",
    "description": "blah blah blah",
    "features": null,
    "url": "url.git",
    "ref": "main"
  }
]`)
		repos, err := catalog.ParseRepos(dummyJSON)
		require.NoError(t, err)

		got, err := repos.AsPlain()
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  blah blah blah

name-of-other-project | url.git | main
  blah blah blah

`
		assert.Equal(t, want, got)
	})

	t.Run("ignores features when none present", func(t *testing.T) {
		dummyJSON := []byte(`[
  {
    "id": "name-of-project",
    "description": "blah blah blah",
    "features": null,
    "url": "url.git",
    "ref": "main"
  }
]`)
		repos, err := catalog.ParseRepos(dummyJSON)
		require.NoError(t, err)

		got, err := repos.AsPlain()
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  blah blah blah

`
		assert.Equal(t, want, got)
	})

	t.Run("includes features when present", func(t *testing.T) {
		dummyJSON := []byte(`[
  {
    "id": "name-of-project",
    "description": "blah blah blah",
    "features": ["walnut", "almond"],
    "url": "url.git",
    "ref": "main"
  }
]`)
		repos, err := catalog.ParseRepos(dummyJSON)
		require.NoError(t, err)

		got, err := repos.AsPlain()
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  blah blah blah

`
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("correctly wraps new lines in the description", func(t *testing.T) {
		dummyJSON := []byte(`[
  {
    "id": "name-of-project",
    "description": "This sentence exists purely to verify that text wrapping behaves correctly when the content is long enough to span multiple lines.",
    "features": ["walnut", "almond"],
    "url": "url.git",
    "ref": "main"
  }
]`)
		repos, err := catalog.ParseRepos(dummyJSON)
		require.NoError(t, err)

		got, err := repos.AsPlain()
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  This sentence exists purely to verify that text wrapping behaves correctly
  when the content is long enough to span multiple lines.

`
		assert.Equal(t, want, got)
	})

	t.Run("correctly splits paragraphs in the description", func(t *testing.T) {
		dummyJSON := []byte(`[
  {
    "id": "name-of-project",
    "description": "blah blah blah\n\nblah blah blah",
    "features": ["walnut", "almond"],
    "url": "url.git",
    "ref": "main"
  }
]`)
		repos, err := catalog.ParseRepos(dummyJSON)
		require.NoError(t, err)

		got, err := repos.AsPlain()
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  blah blah blah

  blah blah blah

`
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("correctly prints to json", func(t *testing.T) {
		dummyJSON := []byte(`[
  {
    "id": "name-of-project",
    "description": "blah blah blah\n\nblah blah blah",
    "features": ["walnut", "almond"],
    "url": "url.git",
    "ref": "main"
  }
]`)
		repos, err := catalog.ParseRepos(dummyJSON)
		require.NoError(t, err)

		got, err := repos.AsJSON()
		require.NoError(t, err)

		var wantObj any
		var gotObj any
		require.NoError(t, json.Unmarshal(dummyJSON, &wantObj))
		require.NoError(t, json.Unmarshal([]byte(got), &gotObj))

		assert.Equal(t, wantObj, gotObj)
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
