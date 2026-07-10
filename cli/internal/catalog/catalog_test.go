package catalog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/arm/topo/cli/internal/catalog"
	"github.com/arm/topo/cli/internal/deploy/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProjectsFromURL(t *testing.T) {
	t.Run("given a remote url, it fetches the catalog json", func(t *testing.T) {
		projects := []catalog.Project{validProject()}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/give-json" {
				w.Write(asJSON(projects)) // nolint:errcheck
			}
		}))
		defer server.Close()

		url := fmt.Sprintf("%s/give-json", server.URL)
		got, err := catalog.ListProjectsFromURL(context.Background(), url)

		require.NoError(t, err)
		assert.Equal(t, projects, got)
	})

	t.Run("given a file:// url, it fetches the catalog json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.json")
		projects := []catalog.Project{validProject()}
		testutil.RequireWriteFile(t, path, string(asJSON(projects)))

		url := fmt.Sprintf("file://%s", path)
		got, err := catalog.ListProjectsFromURL(context.Background(), url)

		require.NoError(t, err)
		assert.Equal(t, projects, got)
	})

	t.Run("errors when payload doesn't validate against schema", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.json")
		projects := []catalog.Project{{Name: "aloha"}}
		testutil.RequireWriteFile(t, path, string(asJSON(projects)))

		url := fmt.Sprintf("file://%s", path)
		_, err := catalog.ListProjectsFromURL(context.Background(), url)

		require.Error(t, err)
	})

	t.Run("errors when request fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer server.Close()

		url := fmt.Sprintf("%s/give-json-pretty-please", server.URL)
		_, err := catalog.ListProjectsFromURL(context.Background(), url)

		assert.Error(t, err)
	})

	t.Run("errors for invalid JSON", func(t *testing.T) {
		jsonData := []byte(`[{"id": "test", invalid}]`)
		path := filepath.Join(t.TempDir(), "file.json")
		testutil.RequireWriteFile(t, path, string(jsonData))

		url := fmt.Sprintf("file://%s", path)
		_, err := catalog.ListProjectsFromURL(context.Background(), url)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to unmarshal projects")
	})
}

func asJSON(projects []catalog.Project) []byte {
	data, err := json.Marshal(struct {
		Projects []catalog.Project `json:"projects"`
	}{
		Projects: projects,
	})
	if err != nil {
		panic(err)
	}
	return data
}

func validProject() catalog.Project {
	return catalog.Project{
		Name:        "hi",
		Description: "desc",
		URL:         "https://example.com/project.git",
		Ref:         "main",
	}
}
