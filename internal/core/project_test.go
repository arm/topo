package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const emptyComposeProject = `
name: example-project
services: {}
`

func writeComposeFile(t *testing.T, dir, content string) string {
	t.Helper()
	composePath := filepath.Join(dir, DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(content), 0644), "failed to write compose file")
	return composePath
}

func TestRunInitProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, RunInitProject(dir, "proj", testutil.TestSshTarget))
	composeFile := filepath.Join(dir, "proj", DefaultComposeFileName)
	data, err := os.ReadFile(composeFile)
	require.NoError(t, err)
	var p types.Project
	require.NoError(t, yaml.Unmarshal(data, &p))
	assert.Equal(t, "proj", p.Name)
}

func TestAddService(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		mockGetTemplate := func(id string) (*template.ServiceTemplateRepo, error) {
			if id == "test-template" {
				return &template.ServiceTemplateRepo{Id: "test-template", Url: "https://github.com/example/test-template.git"}, nil
			}
			return nil, fmt.Errorf("template with id %q not found", id)
		}

		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		var calls []struct{ URL, Dest string }
		mockCloner := func(url, dest string) error {
			calls = append(calls, struct{ URL, Dest string }{url, dest})
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			topoService := `
name: "test-service"
description: "Test service"
`
			return os.WriteFile(filepath.Join(dest, template.TopoServiceFilename), []byte(topoService), 0644)
		}

		require.NoError(t, RunAddService(targetProjectFile, "test-template", "test", mockCloner, mockGetTemplate))
		require.Len(t, calls, 1, "expected 1 clone call")
		assert.Equal(t, "https://github.com/example/test-template.git", calls[0].URL)

		data, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err, "failed to read compose file")
		var project types.Project
		require.NoError(t, yaml.Unmarshal(data, &project))
		assert.Contains(t, project.Services, "test")
		assert.Len(t, project.Services, 1)
	})

	t.Run("errors when directory exists", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockFinder := func(id string) (*template.ServiceTemplateRepo, error) {
			return &template.ServiceTemplateRepo{Id: id, Url: "https://github.com/example/test-template.git"}, nil
		}

		conflictDir := filepath.Join(dir, "test")
		require.NoError(t, os.MkdirAll(conflictDir, 0755), "failed to create conflict directory")

		mockCloner := func(url, dest string) error {
			t.Fatal("cloner should not be called when directory exists")
			return nil
		}

		err := RunAddService(targetProjectFile, "test-template", "test", mockCloner, mockFinder)

		require.Error(t, err, "expected error when directory exists")
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("registers named volumes but passes through all volume types", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockFinder := func(id string) (*template.ServiceTemplateRepo, error) {
			return &template.ServiceTemplateRepo{Id: id, Url: "https://example.com/template.git"}, nil
		}

		mockCloner := func(url, dest string) error {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			topoService := `
name: "test-service"
service:
  volumes:
    - "data:/data"
    - "/host:/host"`
			return os.WriteFile(filepath.Join(dest, template.TopoServiceFilename), []byte(topoService), 0644)
		}

		require.NoError(t, RunAddService(targetProjectFile, "test-template", "test", mockCloner, mockFinder))

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)

		want := `
name: example-project
services: 
  test:
    build:
      context: ./test
    volumes:
      - type: volume
        source: data
        target: /data
        volume: {}
      - type: bind
        source: /host
        target: /host
        bind:
          create_host_path: true
volumes:
  data: {}
`
		assert.YAMLEq(t, want, string(got))
	})
}

func TestRunRemoveService(t *testing.T) {
	dir := t.TempDir()
	compose := `name: example-project
services:
  removeMe:
    build:
      context: ./removeMe
`
	targetProjectFile := writeComposeFile(t, dir, compose)
	require.NoError(t, RunRemoveService(targetProjectFile, "removeMe"))
	data, err := os.ReadFile(targetProjectFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "removeMe")
}

func TestGenerateMakefile(t *testing.T) {
	dir := t.TempDir()
	targetProjectFile := filepath.Join(dir, "compose.topo.yaml")
	require.NoError(t, os.WriteFile(targetProjectFile, []byte("name: test"), 0644))
	require.NoError(t, GenerateMakefile(targetProjectFile, testutil.TestSshTarget))
	content, err := os.ReadFile(filepath.Join(dir, "Makefile"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "COMPOSE_FILE    ?= compose.topo.yaml")
}
