package vscode_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/vscode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintConfigMetadata(t *testing.T) {
	var buf bytes.Buffer

	err := vscode.PrintConfigMetadata(&buf)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `boards`)
}
