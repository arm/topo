package service

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintTemplateRepos(t *testing.T) {
	t.Run("prints templates as JSON", func(t *testing.T) {
		var buf bytes.Buffer

		err := PrintTemplateRepos(&buf)

		require.NoError(t, err)
		var templates []TemplateRepo
		require.NoError(t, json.Unmarshal(buf.Bytes(), &templates))
		assert.NotEmpty(t, templates)
	})
}

func TestGetTemplateRepo(t *testing.T) {
	t.Run("when template exists it is found", func(t *testing.T) {
		template, err := GetTemplateRepo("kleidi-llm")

		require.NoError(t, err)
		assert.Equal(t, "kleidi-llm", template.Id)
		assert.NotEmpty(t, template.Url)
	})

	t.Run("when template does not exist, it errors", func(t *testing.T) {
		_, err := GetTemplateRepo("nonexistent-template")

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent-template" not found`)
	})
}
