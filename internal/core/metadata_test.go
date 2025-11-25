package core

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintConfigMetadata(t *testing.T) {
	var buf bytes.Buffer

	err := PrintConfigMetadata(&buf)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), `boards`)
}
