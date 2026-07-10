package vscode_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/arm/topo/internal/vscode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintProject(t *testing.T) {
	compose := `name: demo
services: {}`
	composePath := testutil.WriteComposeFile(t, t.TempDir(), compose)
	var buf bytes.Buffer

	err := vscode.PrintProject(&buf, composePath)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name": "demo"`)
}
