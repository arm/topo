package project_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/template"
	"github.com/arm/topo/internal/testutil"
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

func mockTemplateSourceWithContent(t *testing.T, content, sourceName string) *mockTemplateSource {
	mockSource := &mockTemplateSource{}
	mockSource.On("CopyTo", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		destDir := args.String(0)
		require.NoError(t, os.MkdirAll(destDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(destDir, template.ComposeFilename), []byte(content), 0o644))
	})
	mockSource.On("GetName").Return(sourceName, nil)
	t.Cleanup(func() {
		mockSource.AssertExpectations(t)
	})
	return mockSource
}

func mockTemplateSourceWithErrorOnCopy(t *testing.T, errToReturn error, sourceName string) *mockTemplateSource {
	mockSource := &mockTemplateSource{}
	mockSource.On("CopyTo", mock.Anything).Return(errToReturn)
	mockSource.On("GetName").Return(sourceName, nil)
	t.Cleanup(func() {
		mockSource.AssertExpectations(t)
	})
	return mockSource
}

func TestExtend(t *testing.T) {
	t.Run("extends service from TemplateSource", func(t *testing.T) {
		dir := t.TempDir()
		sourceName := "test-template"
		projectYAML := `
name: example-project
services: {}
`
		targetProjectFile := testutil.WriteComposeFile(t, dir, projectYAML)
		mockSource := mockTemplateSourceWithContent(t, `
services:
  app:
    image: nginx:alpine
  app2:
    image: redis:alpine2

x-topo:
  name: "test-service"
  description: "Test service"
`, sourceName)
		argProvider := arguments.NewStrictProviderChain()

		_, err := project.Extend(targetProjectFile, mockSource, argProvider)
		require.NoError(t, err)

		data, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err, "failed to read compose file")
		sourcePath := filepath.Join(sourceName, "compose.yaml")
		wantYAML := fmt.Sprintf(`
name: example-project
services:
  app:
    extends:
      file: %[1]s
      service: app
  app2:
    extends:
      file: %[1]s
      service: app2
`, sourcePath)
		assert.YAMLEq(t, wantYAML, string(data))
	})

	t.Run("errors when directory exists", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)
		sourceName := "test"
		destDir := filepath.Join(dir, sourceName)
		mockSource := mockTemplateSourceWithErrorOnCopy(t, template.DestDirExistsError{Dir: destDir}, sourceName)
		provider := arguments.NewStrictProviderChain()

		_, err := project.Extend(targetProjectFile, mockSource, provider)

		require.ErrorContains(t, err, "already exists", "expected error when directory exists")
	})

	t.Run("registers named volumes", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)
		sourceName := "ginger"
		mockSource := mockTemplateSourceWithContent(t, `
services:
  app:
    volumes:
      - "pretty_data:/data"
      - "/host:/host"

x-topo:
  name: "ginger-service"
`, sourceName)
		argProvider := arguments.NewStrictProviderChain()

		_, err := project.Extend(targetProjectFile, mockSource, argProvider)
		require.NoError(t, err)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)
		sourcePath := filepath.Join(sourceName, "compose.yaml")
		want := fmt.Sprintf(`
name: example-project
services:
  app:
    extends:
      file: %s
      service: app
volumes:
  pretty_data: {}
`, sourcePath)
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("collects and injects build arguments", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)
		sourceName := "piggy-service"
		mockSource := mockTemplateSourceWithContent(t, `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "piggy-service"
  args:
    GREETING:
      description: "The greeting message"
      required: true
      example: "Hello"
`, sourceName)
		provider := arguments.NewStaticProvider(arguments.ResolvedArg{Name: "GREETING", Value: "Hello, World"})

		_, err := project.Extend(targetProjectFile, mockSource, provider)
		require.NoError(t, err)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)
		sourcePath := filepath.Join(sourceName, "compose.yaml")
		want := fmt.Sprintf(`
name: example-project
services:
  app:
    extends:
      file: %s
      service: app
    build:
      args:
        GREETING: "Hello, World"
`, sourcePath)
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("does not collect optional arguments into x-topo", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)
		sourceName := "oyster-service"
		mockSource := mockTemplateSourceWithContent(t, `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "oyster-service"
  args:
    GREETING:
      description: "The greeting message"
      required: true
      example: "Hello"
    SMALLTALK:
      description: "The small talk message"
      example: "How are you?"
`, sourceName)
		provider := arguments.NewStaticProvider(arguments.ResolvedArg{Name: "GREETING", Value: "Hello, World"})

		_, err := project.Extend(targetProjectFile, mockSource, provider)
		require.NoError(t, err)

		got, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)
		sourcePath := filepath.Join(sourceName, "compose.yaml")
		want := fmt.Sprintf(`
name: example-project
services:
  app:
    extends:
      file: %s
      service: app
    build:
      args:
        GREETING: "Hello, World"
`, sourcePath)
		assert.YAMLEq(t, want, string(got))
	})

	t.Run("cleans up service directory when argument collection fails ", func(t *testing.T) {
		dir := t.TempDir()
		targetProjectFile := testutil.WriteComposeFile(t, dir, emptyComposeProject)

		sourceName := "vinegar-service"
		mockSource := mockTemplateSourceWithContent(t, `
services:
  app:
    image: nginx:alpine

x-topo:
  name: "vinegar-service"
  args:
    GREETING:
      description: "The greeting message"
      required: true
`, sourceName)
		provider := arguments.NewErrorProvider(errors.New("user cancelled"))

		_, err := project.Extend(targetProjectFile, mockSource, provider)

		assert.EqualError(t, err, "user cancelled")
		copiedTemplateDir := filepath.Join(filepath.Dir(targetProjectFile), sourceName)
		_, err = os.Stat(copiedTemplateDir)
		assert.True(t, os.IsNotExist(err), "service directory should be cleaned up after failure")
	})
}

func TestResolveAndApplyArgs(t *testing.T) {
	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		argProvider := arguments.NewStrictProviderChain()

		_, err := project.ResolveAndApplyArgs(invalidPath, argProvider)

		require.ErrorContains(t, err, "can't read compose file")
	})

	t.Run("updates the compose file with resolved arguments", func(t *testing.T) {
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

		_, err := project.ResolveAndApplyArgs(composeFilePath, argProvider)
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
	t.Run("removes specified service from compose file", func(t *testing.T) {
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
		want := `name: example-project
services: {}
`
		assert.YAMLEq(t, want, string(data))
	})

	t.Run("preserves comments when a service is removed", func(t *testing.T) {
		dir := t.TempDir()
		compose := `name: example-project
services:
  removeMe:
    build:
      context: ./removeMe
  # This is a comment that should be preserved
  keepMe:
    build:
      context: ./keepMe
`
		targetProjectFile := testutil.WriteComposeFile(t, dir, compose)

		require.NoError(t, project.RemoveService(targetProjectFile, "removeMe"))

		data, err := os.ReadFile(targetProjectFile)
		require.NoError(t, err)
		want := `name: example-project
services:
  # This is a comment that should be preserved
  keepMe:
    build:
      context: ./keepMe
`
		assert.Equal(t, want, string(data))
	})
}
