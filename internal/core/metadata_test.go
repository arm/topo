package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProject(t *testing.T) {
	dir := t.TempDir()
	compose := `name: demo
services: {}`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(compose), 0644))
	out := testutil.CaptureOutput(func() { GetProject(composePath) })
	assert.Contains(t, out, "\"name\": \"demo\"")
}

func TestGetConfigMetadata(t *testing.T) {
	out := testutil.CaptureOutput(func() {
		require.NoError(t, GetConfigMetadata())
	})
	assert.Contains(t, out, "boards")
}
