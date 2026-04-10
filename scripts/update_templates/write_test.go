package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTemplates(t *testing.T) {
	t.Run("writes json", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "templates.json")
		input := []Template{{
			Name:        "repo",
			Description: "Desc",
			Features:    []string{"SME", "NEON"},
			URL:         "ssh://example",
			Ref:         "main",
		}}

		err := WriteTemplates(path, input)
		require.NoError(t, err)

		raw := testutil.RequireReadFile(t, path)
		var decoded []Template
		require.NoError(t, json.Unmarshal([]byte(raw), &decoded))
		assert.Equal(t, input, decoded)
	})
}
