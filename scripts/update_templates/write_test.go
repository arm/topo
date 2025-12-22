package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteTemplates(t *testing.T) {
	t.Run("writes json", func(t *testing.T) {
		tmp := t.TempDir()
		path := filepath.Join(tmp, "templates.json")
		input := []Template{{
			ID:          "repo",
			Description: "Desc",
			Features:    []string{"SME", "NEON"},
			URL:         "ssh://example",
			Ref:         "main",
		}}

		err := WriteTemplates(path, input)
		require.NoError(t, err)

		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var decoded []Template
		require.NoError(t, json.Unmarshal(raw, &decoded))
		assert.Equal(t, input, decoded)
	})
}
