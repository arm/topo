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
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Url:         "url.git",
				Ref:         "main",
			},
			{
				Id:          "name-of-other-project",
				Description: "blah blah blah",
				Url:         "url.git",
				Ref:         "main",
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
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Url:         "url.git",
				Ref:         "main",
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
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
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
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "This sentence exists purely to verify that text wrapping behaves correctly when the content is long enough to span multiple lines.",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
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
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah\n\nblah blah blah",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
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
		repos := []catalog.Repo{
			{
				Id:          "name-of-project",
				Description: "blah blah blah\n\nblah blah blah",
				Features:    []string{"walnut", "almond"},
				Url:         "url.git",
				Ref:         "main",
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
				"id":          "name-of-project",
				"description": "blah blah blah\n\nblah blah blah",
				"features":    []any{"walnut", "almond"},
				"url":         "url.git",
				"ref":         "main",
			},
		}

		assert.Equal(t, want, got)
	})
}
