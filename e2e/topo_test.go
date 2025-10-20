package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm-debug/topo-cli/configs"
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

func TestIntegration_ListTemplates(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "list-templates")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal(out, &arr))
	assert.NotEmpty(t, arr)
}

func TestIntegration_Version(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "version")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(configs.VersionTxt), strings.TrimSpace(string(out)))
}

func TestIntegration_UnknownCommand(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "no-such-command")
	out, err := cmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(out), "no-such-command")
}
