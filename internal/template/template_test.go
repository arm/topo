package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Run("lists templates to stdout", func(t *testing.T) {
		out := testutil.CaptureOutput(func() {
			require.NoError(t, List())
		})

		var arr []ServiceTemplateRepo
		require.NoError(t, json.Unmarshal([]byte(out), &arr))
		assert.NotEmpty(t, arr)
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
