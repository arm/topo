package vscode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/arm-debug/topo-cli/internal/vscode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintProject(t *testing.T) {
	compose := `name: demo
services: {}`
	composePath := filepath.Join(t.TempDir(), project.ComposeFilename)
	require.NoError(t, os.WriteFile(composePath, []byte(compose), 0o644))
	var buf bytes.Buffer

	err := vscode.PrintProject(&buf, composePath)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name": "demo"`)
}
