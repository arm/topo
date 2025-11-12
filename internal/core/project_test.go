package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/arm-debug/topo-cli/internal/testutil"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const emptyComposeProject = `
name: example-project
services: {}
`

type mockServiceSource struct {
	mock.Mock
}

func (m *mockServiceSource) CopyTo(destDir string) error {
	args := m.Called(destDir)
	return args.Error(0)
}

func (m *mockServiceSource) String() string {
	args := m.Called()
	return args.String(0)
}

func writeComposeFile(t *testing.T, dir, content string) string {
	t.Helper()
	composePath := filepath.Join(dir, DefaultComposeFileName)
	require.NoError(t, os.WriteFile(composePath, []byte(content), 0644), "failed to write compose file")
	return composePath
}

func TestInitProject(t *testing.T) {
	t.Run("creates an empty compose file at the given location", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, InitProject(dir, testutil.TestSshTarget))

		composeFile := filepath.Join(dir, DefaultComposeFileName)
		data, err := os.ReadFile(composeFile)
		require.NoError(t, err)
		var p types.Project
		require.NoError(t, yaml.Unmarshal(data, &p))
		assert.Empty(t, p.Services)
	})
}

func TestAddService(t *testing.T) {
	t.Run("adds service from ServiceSource", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockServiceSource{}
		destDir := filepath.Join(dir, "test")

		mockSource.On("CopyTo", destDir).Return(nil).Run(func(args mock.Arguments) {
			dest := args.String(0)
			require.NoError(t, os.MkdirAll(dest, 0755))
			topoService := `
name: "test-service"
description: "Test service"
`
			require.NoError(t, os.WriteFile(filepath.Join(dest, service.TopoServiceFilename), []byte(topoService), 0644))
		})

		require.NoError(t, AddService(targetProjectFile, "test", mockSource))

		mockSource.AssertExpectations(t)

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

		conflictDir := filepath.Join(dir, "test")
		require.NoError(t, os.MkdirAll(conflictDir, 0755), "failed to create conflict directory")

		mockSource := &mockServiceSource{}

		err := AddService(targetProjectFile, "test", mockSource)

		require.Error(t, err, "expected error when directory exists")
		assert.Contains(t, err.Error(), "already exists")
		mockSource.AssertNotCalled(t, "CopyTo")
	})

	t.Run("registers named volumes but passes through all volume types", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockServiceSource{}
		destDir := filepath.Join(dir, "test")

		mockSource.On("CopyTo", destDir).Return(nil).Run(func(args mock.Arguments) {
			dest := args.String(0)
			require.NoError(t, os.MkdirAll(dest, 0755))
			topoService := `
name: "test-service"
service:
  volumes:
    - "data:/data"
    - "/host:/host"`
			require.NoError(t, os.WriteFile(filepath.Join(dest, service.TopoServiceFilename), []byte(topoService), 0644))
		})

		require.NoError(t, AddService(targetProjectFile, "test", mockSource))

		mockSource.AssertExpectations(t)

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

func TestRemoveService(t *testing.T) {
	dir := t.TempDir()
	compose := `name: example-project
services:
  removeMe:
    build:
      context: ./removeMe
`
	targetProjectFile := writeComposeFile(t, dir, compose)
	require.NoError(t, RemoveService(targetProjectFile, "removeMe"))
	data, err := os.ReadFile(targetProjectFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "removeMe")
}
