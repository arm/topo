package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRunInitProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, RunInitProject(dir, "proj", TestSshTarget))
	composeFile := filepath.Join(dir, "proj", DefaultComposeFileName)
	data, err := os.ReadFile(composeFile)
	require.NoError(t, err)
	var p Project
	require.NoError(t, yaml.Unmarshal(data, &p))
	assert.Equal(t, "proj", p.Name)
}

func TestAddServiceWithNoRuntime(t *testing.T) {
	dir := t.TempDir()
	compose := `name: example-project
services:
  ambient-zephyr:
    build:
      context: ./ambient-zephyr
    runtime: io.containerd.remoteproc.v1
    annotations:
      remoteproc.mcu: imx-rproc
`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(compose), 0644))
	var calls []struct{ URL, Dest string }
	mockCloner := func(url, dest string) error { calls = append(calls, struct{ URL, Dest string }{url, dest}); return nil }
	require.NoError(t, RunAddService(composePath, "cortexa-welcome", "test", mockCloner))
	require.Len(t, calls, 1)
	assert.Equal(t, "https://github.com/Arm-Debug/topo-cortexa-welcome", calls[0].URL)
}

func TestRunRemoveService(t *testing.T) {
	dir := t.TempDir()
	compose := `name: example-project
services:
  removeMe:
    build:
      context: ./removeMe
`
	composePath := filepath.Join(dir, DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(compose), 0644))
	require.NoError(t, RunRemoveService(composePath, "removeMe"))
	data, err := os.ReadFile(composePath)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "removeMe")
}

func TestGenerateMakefile(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "compose.topo.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte("name: test"), 0644))
	require.NoError(t, GenerateMakefile(composePath, TestSshTarget))
	content, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "COMPOSE_FILE    ?= compose.topo.yaml")
}
