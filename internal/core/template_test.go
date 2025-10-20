package core

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplatesOperation(t *testing.T) {
	out := captureOutput(func() {
		require.NoError(t, ListTemplates())
	})
	var arr []Template
	require.NoError(t, json.Unmarshal([]byte(out), &arr))
	assert.NotEmpty(t, arr)
}
