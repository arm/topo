package project_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/arm-debug/topo-cli/internal/source"
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

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Provide(args []arguments.Arg) (map[string]string, error) {
	callArgs := m.Called(args)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(map[string]string), callArgs.Error(1)
}

func (m *mockProvider) Name() string {
	return "mock"
}

func writeComposeFile(t *testing.T, dir, content string) string {
	t.Helper()
	composePath := filepath.Join(dir, project.ComposeFilename)
	require.NoError(t, os.WriteFile(composePath, []byte(content), 0o644), "failed to write compose file")
	return composePath
}

func TestInit(t *testing.T) {
	t.Run("creates an empty compose file at the given location", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, project.Init(dir))

		composeFile := filepath.Join(dir, project.ComposeFilename)
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
			require.NoError(t, os.MkdirAll(dest, 0o755))
			composeFileContents := `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "test-service"
  description: "Test service"
`
			require.NoError(t, os.WriteFile(filepath.Join(dest, service.ComposeServiceFilename), []byte(composeFileContents), 0o644))
		})
		collector := arguments.NewCollector()

		require.NoError(t, project.AddService(targetProjectFile, "test", mockSource, collector))

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

		destDir := filepath.Join(dir, "test")

		mockSource := &mockServiceSource{}
		mockSource.On("CopyTo", destDir).Return(source.DestDirExistsError{Dir: destDir})
		collector := arguments.NewCollector()

		err := project.AddService(targetProjectFile, "test", mockSource, collector)

		require.Error(t, err, "expected error when directory exists")
		assert.Contains(t, err.Error(), "already exists")
		mockSource.AssertExpectations(t)
	})

	t.Run("registers named volumes", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockServiceSource{}
		destDir := filepath.Join(dir, "test")

		mockSource.On("CopyTo", destDir).Return(nil).Run(func(args mock.Arguments) {
			dest := args.String(0)
			require.NoError(t, os.MkdirAll(dest, 0o755))
			composeFileContents := `
services:
  app:
    volumes:
      - "pretty_data:/data"
      - "/host:/host"

x-topo:
  name: "test-service"
`
			require.NoError(t, os.WriteFile(filepath.Join(dest, service.ComposeServiceFilename), []byte(composeFileContents), 0o644))
		})
		collector := arguments.NewCollector()

		require.NoError(t, project.AddService(targetProjectFile, "test", mockSource, collector))

		mockSource.AssertExpectations(t)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)

		want := `
name: example-project
services:
  test:
    extends:
      file: ./test/compose.service.yaml
      service: app
volumes:
  pretty_data: {}
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("collects and injects build arguments", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockServiceSource{}
		provider := &mockProvider{}
		destDir := filepath.Join(dir, "test")

		mockSource.On("CopyTo", destDir).Return(nil).Run(func(args mock.Arguments) {
			dest := args.String(0)
			require.NoError(t, os.MkdirAll(dest, 0o755))
			composeFileContents := `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "test-service"
  args:
    GREETING:
      description: "The greeting message"
      required: true
      example: "Hello"
`
			require.NoError(t, os.WriteFile(filepath.Join(dest, service.ComposeServiceFilename), []byte(composeFileContents), 0o644))
		})

		expectedArgs := []arguments.Arg{
			{
				Name:        "GREETING",
				Description: "The greeting message",
				Required:    true,
				Example:     "Hello",
			},
		}
		provider.On("Provide", expectedArgs).Return(map[string]string{"GREETING": "Hello, World"}, nil)
		collector := arguments.NewCollector(provider)

		require.NoError(t, project.AddService(targetProjectFile, "test", mockSource, collector))

		mockSource.AssertExpectations(t)
		provider.AssertExpectations(t)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)

		want := `
name: example-project
services:
  test:
    extends:
      file: ./test/compose.service.yaml
      service: app
    build:
      args:
        GREETING: "Hello, World"
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("cleans up service directory when argument collection fails", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := writeComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockServiceSource{}
		provider := &mockProvider{}
		destDir := filepath.Join(dir, "test")

		mockSource.On("CopyTo", destDir).Return(nil).Run(func(args mock.Arguments) {
			dest := args.String(0)
			require.NoError(t, os.MkdirAll(dest, 0o755))
			composeFileContents := `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "test-service"
  args:
    GREETING:
      description: "The greeting message"
      required: true
`
			require.NoError(t, os.WriteFile(filepath.Join(dest, service.ComposeServiceFilename), []byte(composeFileContents), 0o644))
		})

		provider.On("Provide", mock.Anything).Return(nil, fmt.Errorf("user cancelled"))
		collector := arguments.NewCollector(provider)

		err := project.AddService(targetProjectFile, "test", mockSource, collector)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "user cancelled")

		_, err = os.Stat(destDir)
		assert.True(t, os.IsNotExist(err), "service directory should be cleaned up after failure")

		mockSource.AssertExpectations(t)
		provider.AssertExpectations(t)
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
	require.NoError(t, project.RemoveService(targetProjectFile, "removeMe"))
	data, err := os.ReadFile(targetProjectFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "removeMe")
}
