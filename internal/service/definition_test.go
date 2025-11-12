package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDefinition(t *testing.T) {
	t.Run("returns valid service template manifest when one exists", func(t *testing.T) {
		dir := t.TempDir()

		topoService := `
name: "test-service"
description: "Test service"
`
		os.WriteFile(filepath.Join(dir, TopoServiceFilename), []byte(topoService), 0644)

		got, err := ParseDefinition(dir)
		require.NoError(t, err)

		assert.Equal(t, "test-service", got.Name)
		assert.Equal(t, "Test service", got.Description)
	})

	t.Run("errors when topo-service.yaml missing", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ParseDefinition(dir)
		require.Errorf(t, err, "expected error when %s is missing", TopoServiceFilename)
		assert.Contains(t, err.Error(), TopoServiceFilename)
	})
}
