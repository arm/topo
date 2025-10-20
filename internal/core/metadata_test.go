package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProject(t *testing.T) {
	dir := t.TempDir()
	compose := `name: demo
services: {}`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(compose), 0644))
	out := captureOutput(func() { GetProject(composePath) })
	assert.Contains(t, out, "\"name\": \"demo\"")
}

func TestGetConfigMetadata(t *testing.T) {
	out := captureOutput(func() {
		require.NoError(t, GetConfigMetadata())
	})
	assert.Contains(t, out, "boards")
}
