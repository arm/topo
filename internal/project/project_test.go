package project_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/project"
	"github.com/arm-debug/topo-cli/internal/template"
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

type mockTemplateSource struct {
	mock.Mock
}

func (m *mockTemplateSource) CopyTo(destDir string) error {
	args := m.Called(destDir)
	return args.Error(0)
}

func (m *mockTemplateSource) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockTemplateSource) GetName() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func TestInit(t *testing.T) {
	t.Run("creates an empty compose file at the given location", func(t *testing.T) {
		dir := t.TempDir()

		require.NoError(t, project.Init(dir))

		composeFile := filepath.Join(dir, template.ComposeFilename)
		data, err := os.ReadFile(composeFile)
		require.NoError(t, err)
		var p types.Project
		require.NoError(t, yaml.Unmarshal(data, &p))
		assert.Empty(t, p.Services)
	})
}

func TestExtend(t *testing.T) {
	t.Run("extends service from TemplateSource", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockTemplateSource{}
		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), "test")
		mockSource.On("CopyTo", copiedTemplateDir).Return(nil).Run(func(args mock.Arguments) {
			require.NoError(t, os.MkdirAll(copiedTemplateDir, 0o755))
			composeFileContents := `
services:
  app:
    image: nginx:alpine
  app2:
    image: redis:alpine2

x-topo:
  name: "test-service"
  description: "Test service"
`
			require.NoError(t, os.WriteFile(filepath.Join(copiedTemplateDir, template.ComposeFilename), []byte(composeFileContents), 0o644))
		})
		mockSource.On("GetName").Return("test", nil)

		argProvider := arguments.NewStrictProviderChain()

		require.NoError(t, project.Extend(targetProjectFile, mockSource, argProvider))

		mockSource.AssertExpectations(t)

		data, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err, "failed to read compose file")
		var project types.Project
		require.NoError(t, yaml.Unmarshal(data, &project))
		assert.Contains(t, project.Services, "app")
		assert.Contains(t, project.Services, "app2")
		assert.Len(t, project.Services, 2)
	})

	t.Run("errors when directory exists", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		destDir := filepath.Join(dir, "test")

		mockSource := &mockTemplateSource{}
		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), "test")
		mockSource.On("CopyTo", copiedTemplateDir).Return(template.DestDirExistsError{Dir: destDir})
		mockSource.On("GetName").Return("test", nil)
		provider := arguments.NewStrictProviderChain()

		err := project.Extend(targetProjectFile, mockSource, provider)

		require.Error(t, err, "expected error when directory exists")
		assert.Contains(t, err.Error(), "already exists")
		mockSource.AssertExpectations(t)
	})

	t.Run("registers named volumes", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockTemplateSource{}
		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), "test")
		mockSource.On("CopyTo", copiedTemplateDir).Return(nil).Run(func(args mock.Arguments) {
			require.NoError(t, os.MkdirAll(copiedTemplateDir, 0o755))
			composeFileContents := `
services:
  app:
    volumes:
      - "pretty_data:/data"
      - "/host:/host"

x-topo:
  name: "test-service"
`
			require.NoError(t, os.WriteFile(filepath.Join(copiedTemplateDir, template.ComposeFilename), []byte(composeFileContents), 0o644))
		})
		mockSource.On("GetName").Return("test", nil)

		argProvider := arguments.NewStrictProviderChain()

		require.NoError(t, project.Extend(targetProjectFile, mockSource, argProvider))

		mockSource.AssertExpectations(t)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)

		want := `
name: example-project
services:
  app:
    extends:
      file: ./test/compose.yaml
      service: app
volumes:
  pretty_data: {}
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("collects and injects build arguments", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockTemplateSource{}

		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), "test")
		mockSource.On("CopyTo", copiedTemplateDir).Return(nil).Run(func(args mock.Arguments) {
			require.NoError(t, os.MkdirAll(copiedTemplateDir, 0o755))
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
			require.NoError(t, os.WriteFile(filepath.Join(copiedTemplateDir, template.ComposeFilename), []byte(composeFileContents), 0o644))
		})
		mockSource.On("GetName").Return("test", nil)

		provider := arguments.NewStaticProvider(arguments.ResolvedArg{Name: "GREETING", Value: "Hello, World"})

		require.NoError(t, project.Extend(targetProjectFile, mockSource, provider))

		mockSource.AssertExpectations(t)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)

		want := `
name: example-project
services:
  app:
    extends:
      file: ./test/compose.yaml
      service: app
    build:
      args:
        GREETING: "Hello, World"
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("does not collect optional arguments into x-topo", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockTemplateSource{}

		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), "test")
		mockSource.On("CopyTo", copiedTemplateDir).Return(nil).Run(func(args mock.Arguments) {
			require.NoError(t, os.MkdirAll(copiedTemplateDir, 0o755))
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
    SMALLTALK:
      description: "The small talk message"
      example: "How are you?"
`
			require.NoError(t, os.WriteFile(filepath.Join(copiedTemplateDir, template.ComposeFilename), []byte(composeFileContents), 0o644))
		})
		mockSource.On("GetName").Return("test", nil)

		provider := arguments.NewStaticProvider(arguments.ResolvedArg{Name: "GREETING", Value: "Hello, World"})

		require.NoError(t, project.Extend(targetProjectFile, mockSource, provider))

		mockSource.AssertExpectations(t)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)

		want := `
name: example-project
services:
  app:
    extends:
      file: ./test/compose.yaml
      service: app
    build:
      args:
        GREETING: "Hello, World"
`
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("cleans up service directory when argument collection fails ", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		mockSource := &mockTemplateSource{}

		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), "test")
		mockSource.On("CopyTo", copiedTemplateDir).Return(nil).Run(func(args mock.Arguments) {
			require.NoError(t, os.MkdirAll(copiedTemplateDir, 0o755))
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
			require.NoError(t, os.WriteFile(filepath.Join(copiedTemplateDir, template.ComposeFilename), []byte(composeFileContents), 0o644))
		})
		mockSource.On("GetName").Return("test", nil)

		provider := arguments.NewErrorProvider(errors.New("user cancelled"))

		err := project.Extend(targetProjectFile, mockSource, provider)

		require.Error(t, err)
		assert.EqualError(t, err, "user cancelled")

		_, err = os.Stat(copiedTemplateDir)
		assert.True(t, os.IsNotExist(err), "service directory should be cleaned up after failure")

		mockSource.AssertExpectations(t)
	})
}

func TestInitTemplate(t *testing.T) {
	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		argProvider := arguments.NewStrictProviderChain()

		err := project.InitTemplate(invalidPath, argProvider, io.Discard)

		require.ErrorContains(t, err, "error reading project file")
	})

	t.Run("succeeds and writes an updated file", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
services:
  app:
    build:
      context: .
      args:
        FOO: bar

x-topo:
  name: My Project
  args:
    FOO:
      description: a dummy argument
      required: true
      example: bar
`
		composeFilePath := filepath.Join(dir, template.ComposeFilename)
		testutil.RequireWriteFile(t, composeFilePath, composeFileContents)
		provider := arguments.NewStaticProvider(arguments.ResolvedArg{Name: "FOO", Value: "baz"})
		argProvider := arguments.NewStrictProviderChain(provider)

		err := project.InitTemplate(composeFilePath, argProvider, io.Discard)
		require.NoError(t, err)

		want := `
services:
  app:
    build:
      context: .
      args:
        FOO: baz

x-topo:
  name: My Project
  args:
    FOO:
      description: a dummy argument
      required: true
      example: bar
`
		got, err := os.ReadFile(composeFilePath)
		require.NoError(t, err)

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
	targetProjectFile := testutil.WriteComposeFile(t, dir, compose)
	require.NoError(t, project.RemoveService(targetProjectFile, "removeMe"))
	data, err := os.ReadFile(targetProjectFile)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "removeMe")
}
