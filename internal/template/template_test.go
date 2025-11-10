package template

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintList(t *testing.T) {
	t.Run("prints templates as JSON", func(t *testing.T) {
		var buf bytes.Buffer

		err := PrintList(&buf)

		require.NoError(t, err)
		var templates []ServiceTemplateRepo
		require.NoError(t, json.Unmarshal(buf.Bytes(), &templates))
		assert.NotEmpty(t, templates)
	})
}

func TestGetTemplate(t *testing.T) {
	t.Run("when template exists it is found", func(t *testing.T) {
		template, err := GetTemplate("kleidi-llm")

		require.NoError(t, err)
		assert.Equal(t, "kleidi-llm", template.Id)
		assert.NotEmpty(t, template.Url)
	})

	t.Run("when template does not exist, it errors", func(t *testing.T) {
		_, err := GetTemplate("nonexistent-template")

		require.Error(t, err)
		assert.ErrorContains(t, err, `"nonexistent-template" not found`)
	})
}

func TestParseServiceFromTopo(t *testing.T) {
	t.Run("returns valid service template manifest when one exists", func(t *testing.T) {
		dir := t.TempDir()

		topoService := `
name: "test-service"
description: "Test service"
`
		os.WriteFile(filepath.Join(dir, TopoServiceFilename), []byte(topoService), 0644)

		got, err := ParseServiceDefinition(dir)
		require.NoError(t, err)

		assert.Equal(t, "test-service", got.Name)
		assert.Equal(t, "Test service", got.Description)
	})

	t.Run("errors when topo-service.yaml missing", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ParseServiceDefinition(dir)
		require.Errorf(t, err, "expected error when %s is missing", TopoServiceFilename)
		assert.Contains(t, err.Error(), TopoServiceFilename)
	})
}
