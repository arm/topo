package catalog_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/deploy/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplates(t *testing.T) {
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

		templates, err := catalog.ParseTemplates(jsonData)

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

		_, err := catalog.ParseTemplates(jsonData)

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

		_, err := catalog.ParseTemplates(jsonData)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})
}

func TestFetchTemplatesJSON(t *testing.T) {
	t.Run("given a remote url, it fetches the catalog json", func(t *testing.T) {
		catalogJSON := `[{"json": "for-real"}]`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/give-json" {
				w.Write([]byte(catalogJSON))
			}
		}))
		defer server.Close()

		url := fmt.Sprintf("%s/give-json", server.URL)
		got, err := catalog.FetchTemplatesJSON(url)

		require.NoError(t, err)
		assert.Equal(t, catalogJSON, string(got))
	})

	t.Run("given a file:// url, it fetches the catalog json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.json")
		catalogJSON := `[{"json": "actually-json"}]`
		testutil.RequireWriteFile(t, path, catalogJSON)

		url := fmt.Sprintf("file://%s", path)
		got, err := catalog.FetchTemplatesJSON(url)

		require.NoError(t, err)
		assert.Equal(t, catalogJSON, string(got))
	})

	t.Run("returns error when request fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)

		}))
		defer server.Close()

		url := fmt.Sprintf("%s/give-json-pretty-please", server.URL)
		_, err := catalog.FetchTemplatesJSON(url)

		assert.Error(t, err)
	})
}
