package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "topo")
	cmd := exec.Command("go", "build", "-o", bin, "../cmd/topo")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "build failed: %s", out)
	return bin
}

func TestListTemplates(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "list-service-templates")
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
