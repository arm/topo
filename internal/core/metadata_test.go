package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintProject(t *testing.T) {
	compose := `name: demo
services: {}`
	composePath := filepath.Join(t.TempDir(), DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(compose), 0644))
	var buf bytes.Buffer

	err := PrintProject(&buf, composePath)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name": "demo"`)
}

func TestPrintConfigMetadata(t *testing.T) {
	var buf bytes.Buffer

	err := PrintConfigMetadata(&buf)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `boards`)
}
