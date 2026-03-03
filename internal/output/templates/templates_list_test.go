package templates_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintTemplateRepos(t *testing.T) {
	t.Run("prints multiple items correctly", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name:        "name-of-project",
					Description: "blah blah blah",
					URL:         "url.git",
					Ref:         "main",
				},
			},
			{
				Repo: catalog.Repo{
					Name:        "name-of-other-project",
					Description: "blah blah blah",
					URL:         "url.git",
					Ref:         "main",
				},
			},
		}

		var outBuf bytes.Buffer

		err := printable.Print(
			templates.RepoCollection(repos),
			&outBuf,
			term.Plain,
		)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  blah blah blah

name-of-other-project | url.git | main
  blah blah blah

`
		assert.Equal(t, want, outBuf.String())
	})

	t.Run("ignores features when none present", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name:        "name-of-project",
					Description: "blah blah blah",
					URL:         "url.git",
					Ref:         "main",
				},
			},
		}

		var outBuf bytes.Buffer

		err := printable.Print(
			templates.RepoCollection(repos),
			&outBuf,
			term.Plain,
		)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  blah blah blah

`
		assert.Equal(t, want, outBuf.String())
	})

	t.Run("includes features when present", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name:        "name-of-project",
					Description: "blah blah blah",
					Features:    []string{"walnut", "almond"},
					URL:         "url.git",
					Ref:         "main",
				},
			},
		}

		var outBuf bytes.Buffer

		err := printable.Print(
			templates.RepoCollection(repos),
			&outBuf,
			term.Plain,
		)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  blah blah blah

`
		assert.Equal(t, want, outBuf.String())
	})

	t.Run("correctly wraps long descriptions", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name:        "name-of-project",
					Description: "This sentence exists purely to verify that text wrapping behaves correctly when the content is long enough to span multiple lines.",
					Features:    []string{"walnut", "almond"},
					URL:         "url.git",
					Ref:         "main",
				},
			},
		}

		var outBuf bytes.Buffer

		err := printable.Print(
			templates.RepoCollection(repos),
			&outBuf,
			term.Plain,
		)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  This sentence exists purely to verify that text wrapping behaves correctly
  when the content is long enough to span multiple lines.

`
		assert.Equal(t, want, outBuf.String())
	})

	t.Run("correctly splits paragraphs in the description", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name:        "name-of-project",
					Description: "blah blah blah\n\nblah blah blah",
					Features:    []string{"walnut", "almond"},
					URL:         "url.git",
					Ref:         "main",
				},
			},
		}

		var outBuf bytes.Buffer

		err := printable.Print(
			templates.RepoCollection(repos),
			&outBuf,
			term.Plain,
		)
		require.NoError(t, err)

		want := `name-of-project | url.git | main
  Features: walnut, almond
  blah blah blah

  blah blah blah

`
		assert.Equal(t, want, outBuf.String())
	})

	t.Run("correctly prints json", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name:        "name-of-project",
					Description: "blah blah blah\n\nblah blah blah",
					Features:    []string{"walnut", "almond"},
					URL:         "url.git",
					Ref:         "main",
				},
			},
		}

		var outBuf bytes.Buffer

		err := printable.Print(
			templates.RepoCollection(repos),
			&outBuf,
			term.JSON,
		)
		require.NoError(t, err)

		var got any
		require.NoError(t, json.Unmarshal(outBuf.Bytes(), &got))

		want := []any{
			map[string]any{
				"name":        "name-of-project",
				"description": "blah blah blah\n\nblah blah blah",
				"features":    []any{"walnut", "almond"},
				"url":         "url.git",
				"ref":         "main",
			},
		}

		assert.Equal(t, want, got)
	})

	t.Run("prints compatibility marker when supported = true", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name: "name-of-project",
					URL:  "url.git",
					Ref:  "main",
				},
				Compatibility: &catalog.Compatibility{
					Supported: true,
				},
			},
		}

		var outBuf bytes.Buffer
		err := printable.Print(templates.RepoCollection(repos), &outBuf, term.Plain)
		require.NoError(t, err)

		assert.Equal(t, "✅ name-of-project | url.git | main\n\n", outBuf.String())
	})

	t.Run("prints compatibility marker if project is compatible and vice versa", func(t *testing.T) {
		compatibleRepo := catalog.RepoWithCompatibility{
			Repo:          catalog.Repo{Name: "lasagne"},
			Compatibility: &catalog.Compatibility{Supported: true},
		}
		incompatibleRepo := catalog.RepoWithCompatibility{
			Repo:          catalog.Repo{Name: "spaghetti"},
			Compatibility: &catalog.Compatibility{Supported: false},
		}
		repos := []catalog.RepoWithCompatibility{compatibleRepo, incompatibleRepo}

		var outBuf bytes.Buffer
		err := printable.Print(templates.RepoCollection(repos), &outBuf, term.Plain)
		require.NoError(t, err)

		assert.Contains(t, outBuf.String(), "✅ lasagne")
		assert.Contains(t, outBuf.String(), "❌ spaghetti")
	})

	t.Run("json includes compatibility marker if project is compatible and vice versa", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name: "lasagne",
					URL:  "url.git",
					Ref:  "main",
				},
				Compatibility: &catalog.Compatibility{Supported: true},
			},
			{
				Repo: catalog.Repo{
					Name: "spaghetti",
					URL:  "url.git",
					Ref:  "main",
				},
				Compatibility: &catalog.Compatibility{Supported: false},
			},
		}

		var outBuf bytes.Buffer
		err := printable.Print(templates.RepoCollection(repos), &outBuf, term.JSON)
		require.NoError(t, err)

		var got any
		require.NoError(t, json.Unmarshal(outBuf.Bytes(), &got))

		want := []any{
			map[string]any{
				"name":        "lasagne",
				"description": "",
				"features":    nil,
				"url":         "url.git",
				"ref":         "main",
				"compatibility": map[string]any{
					"supported": true,
				},
			},
			map[string]any{
				"name":        "spaghetti",
				"description": "",
				"features":    nil,
				"url":         "url.git",
				"ref":         "main",
				"compatibility": map[string]any{
					"supported": false,
				},
			},
		}

		assert.Equal(t, want, got)
	})

	t.Run("json omits compatibility when not present", func(t *testing.T) {
		repos := []catalog.RepoWithCompatibility{
			{
				Repo: catalog.Repo{
					Name: "name-of-project",
					URL:  "url.git",
					Ref:  "main",
				},
			},
		}

		var outBuf bytes.Buffer
		err := printable.Print(templates.RepoCollection(repos), &outBuf, term.JSON)
		require.NoError(t, err)

		var got any
		require.NoError(t, json.Unmarshal(outBuf.Bytes(), &got))

		want := []any{
			map[string]any{
				"name":        "name-of-project",
				"description": "",
				"features":    nil,
				"url":         "url.git",
				"ref":         "main",
			},
		}

		assert.Equal(t, want, got)
	})
}
