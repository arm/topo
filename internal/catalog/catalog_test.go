package catalog_test

import (
	"context"
	"encoding/json"
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

func TestListTemplatesFromURL(t *testing.T) {
	t.Run("given a remote url, it fetches the catalog json", func(t *testing.T) {
		repos := []catalog.Repo{validRepo()}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/give-json" {
				w.Write(asJSON(repos)) // nolint:errcheck
			}
		}))
		defer server.Close()

		url := fmt.Sprintf("%s/give-json", server.URL)
		got, err := catalog.ListTemplatesFromURL(context.Background(), url)

		require.NoError(t, err)
		assert.Equal(t, repos, got)
	})

	t.Run("given a file:// url, it fetches the catalog json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.json")
		repos := []catalog.Repo{validRepo()}
		testutil.RequireWriteFile(t, path, string(asJSON(repos)))

		url := fmt.Sprintf("file://%s", path)
		got, err := catalog.ListTemplatesFromURL(context.Background(), url)

		require.NoError(t, err)
		assert.Equal(t, repos, got)
	})

	t.Run("errors when template json doesn't validate against schema", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.json")
		repos := []catalog.Repo{{Name: "aloha"}}
		testutil.RequireWriteFile(t, path, string(asJSON(repos)))

		url := fmt.Sprintf("file://%s", path)
		_, err := catalog.ListTemplatesFromURL(context.Background(), url)

		require.Error(t, err)
	})

	t.Run("errors when request fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer server.Close()

		url := fmt.Sprintf("%s/give-json-pretty-please", server.URL)
		_, err := catalog.ListTemplatesFromURL(context.Background(), url)

		assert.Error(t, err)
	})

	t.Run("errors for invalid JSON", func(t *testing.T) {
		jsonData := []byte(`[{"id": "test", invalid}]`)
		path := filepath.Join(t.TempDir(), "file.json")
		testutil.RequireWriteFile(t, path, string(jsonData))

		url := fmt.Sprintf("file://%s", path)
		_, err := catalog.ListTemplatesFromURL(context.Background(), url)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal templates")
	})

	t.Run("errors for unknown fields", func(t *testing.T) {
		jsonData := []byte(`{
			"$schema": "https://topo.arm.com/schemas/templates/1/schema.json",
			"templates": [
				{
					"name": "test",
					"description": "desc",
					"features": [],
					"url": "https://example.com",
					"ref": "main",
					"yolo-swag": "value"
				}
			]
		}`)
		path := filepath.Join(t.TempDir(), "file.json")
		testutil.RequireWriteFile(t, path, string(jsonData))

		url := fmt.Sprintf("file://%s", path)
		_, err := catalog.ListTemplatesFromURL(context.Background(), url)

		require.Error(t, err)
		assert.ErrorContains(t, err, `additional properties 'yolo-swag' not allowed`)
	})
}

func asJSON(repos []catalog.Repo) []byte {
	data, err := json.Marshal(struct {
		Schema    string         `json:"$schema"`
		Templates []catalog.Repo `json:"templates"`
	}{
		Schema:    "https://topo.arm.com/schemas/templates/1/schema.json",
		Templates: repos,
	})
	if err != nil {
		panic(err)
	}
	return data
}

func validRepo() catalog.Repo {
	return catalog.Repo{
		Name:        "hi",
		Description: "desc",
		URL:         "https://example.com/template.git",
		Ref:         "main",
	}
}
