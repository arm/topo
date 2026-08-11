package project_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/arguments"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockProjectSource struct {
	mock.Mock
}

func (m *mockProjectSource) CopyTo(destDir string) error {
	args := m.Called(destDir)
	return args.Error(0)
}

func (m *mockProjectSource) GetName() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func TestClone(t *testing.T) {
	t.Run("prints summary with next steps", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		mockSource := mockSourceWithContent(t, `
services:
  app:
    image: nginx:alpine
`)
		var output bytes.Buffer

		err := project.NewClone(destDir, mockSource, arguments.NewStrictProviderChain()).Run(&output)

		require.NoError(t, err)
		out := output.String()
		assert.Contains(t, out, "Project ready")
		assert.Contains(t, out, fmt.Sprintf("Created in '%s'", destDir))
		assert.Contains(t, out, "cd "+destDir)
		assert.Contains(t, out, "topo deploy")
	})

	t.Run("clones source into destination directory", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		mockSource := mockSourceWithContent(t, `
services:
  app:
    image: nginx:alpine
`)

		err := project.Clone(destDir, mockSource, arguments.NewStrictProviderChain())

		require.NoError(t, err)
		composeFilePath := filepath.Join(destDir, project.ComposeFilename)
		assert.FileExists(t, composeFilePath)
	})

	t.Run("removes destination directory when parameter resolution fails", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		mockSource := mockSourceWithContent(t, `
services:
  app:
    build:
      args:
        GREETING: ${GREETING}
x-topo:
  parameters:
    GREETING:
      description: "Greeting"
      required: true
`)

		err := project.Clone(destDir, mockSource, arguments.NewStrictProviderChain())

		require.Error(t, err)
		_, statErr := os.Stat(destDir)
		assert.True(t, os.IsNotExist(statErr))
	})
}

func mockSourceWithContent(t *testing.T, content string) *mockProjectSource {
	mockSource := &mockProjectSource{}
	mockSource.On("CopyTo", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		destDir := args.String(0)
		testutil.RequireMkdirAll(t, destDir)
		testutil.RequireWriteFile(t, filepath.Join(destDir, project.ComposeFilename), content)
	})
	t.Cleanup(func() {
		mockSource.AssertExpectations(t)
	})
	return mockSource
}

func TestResolveAndApplyArgs(t *testing.T) {
	t.Run("fails due to an nonexistent compose file", func(t *testing.T) {
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "compose.yaml")
		argProvider := arguments.NewStrictProviderChain()

		err := project.ResolveAndApplyArgs(invalidPath, argProvider)

		require.ErrorContains(t, err, "can't read compose file")
	})

	t.Run("updates the compose file with resolved parameters", func(t *testing.T) {
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
  parameters:
    FOO:
      description: a dummy parameter
      required: true
      example: bar
`
		composeFilePath := filepath.Join(dir, project.ComposeFilename)
		testutil.RequireWriteFile(t, composeFilePath, composeFileContents)
		provider := arguments.NewStaticProvider(arguments.ResolvedArg{Name: "FOO", Value: "baz"})
		argProvider := arguments.NewStrictProviderChain(provider)

		err := project.ResolveAndApplyArgs(composeFilePath, argProvider)
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
  parameters:
    FOO:
      description: a dummy parameter
      required: true
      example: bar
`
		got := testutil.RequireReadFile(t, composeFilePath)

		assert.YAMLEq(t, want, got)
	})

	t.Run("leaves skipped optional parameters unchanged", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `services:
  app:
    build:
      args:
        FOO: current
x-topo:
  parameters:
    FOO:
      required: false
      default: default
`
		composeFilePath := filepath.Join(dir, project.ComposeFilename)
		testutil.RequireWriteFile(t, composeFilePath, composeFileContents)
		argProvider := arguments.NewStrictProviderChain(arguments.NewStaticProvider())

		err := project.ResolveAndApplyArgs(composeFilePath, argProvider)

		require.NoError(t, err)
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
	})
}
