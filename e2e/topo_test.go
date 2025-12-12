package e2e

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "templates")
	out, err := cmd.CombinedOutput()

	require.NoError(t, err)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(out, &arr))
	assert.NotEmpty(t, arr)
}

func TestUnknownCommand(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "yolo-swag")
	out, err := cmd.CombinedOutput()

	assert.Error(t, err)
	want := `unknown command "yolo-swag"`
	assert.Contains(t, string(out), want)
}
