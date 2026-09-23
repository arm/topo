package project_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/parameter"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClone(t *testing.T) {
	t.Run("prints summary with next steps", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		mockSource := mockSourceWithComposeFile(t, `
services:
  app:
    image: nginx:alpine
`)
		var output bytes.Buffer

		err := project.Clone(&output, destDir, mockSource, parameter.NewStrictResolverChain(), filepath.Join(destDir, env.DefaultFilename), false)

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
		mockSource := mockSourceWithComposeFile(t, `
services:
  app:
    image: nginx:alpine
`)

		err := project.Clone(t.Output(), destDir, mockSource, parameter.NewStrictResolverChain(), filepath.Join(destDir, env.DefaultFilename), false)

		require.NoError(t, err)
		composeFilePath := filepath.Join(destDir, compose.DefaultFileName())
		assert.FileExists(t, composeFilePath)
	})

	t.Run("preserves current env values and leaves compose file unchanged", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		composeFileContents := `services:
  app:
    build:
      args:
        GREETING: ${GREETING}
  app-2:
    build:
      args:
        GREETING: "goodbye!"
x-topo:
  parameters:
    GREETING:
      required: true
`
		mockSource := mockSourceWithContent(t, map[string]string{
			compose.DefaultFileName(): composeFileContents,
			env.DefaultFilename:       "GREETING=hello\n",
		})
		resolver := parameter.NewInteractiveResolver(strings.NewReader("\n"), &bytes.Buffer{})

		err := project.Clone(t.Output(), destDir, mockSource, parameter.NewStrictResolverChain(resolver), filepath.Join(destDir, env.DefaultFilename), false)

		require.NoError(t, err)
		composeFilePath := filepath.Join(destDir, compose.DefaultFileName())
		assert.Equal(t, composeFileContents, testutil.RequireReadFile(t, composeFilePath))
		testutil.RequireEnvFileValues(t, filepath.Join(destDir, env.DefaultFilename), map[string]string{"GREETING": "hello"})
	})

	t.Run("removes destination directory when parameter configuration fails", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		mockSource := mockSourceWithComposeFile(t, `
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

		err := project.Clone(t.Output(), destDir, mockSource, parameter.NewStrictResolverChain(), filepath.Join(destDir, env.DefaultFilename), false)

		require.ErrorContains(t, err, "missing value(s) for required parameters")
		_, statErr := os.Stat(destDir)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("rejects parameters that require migration", func(t *testing.T) {
		destDir := filepath.Join(t.TempDir(), "demo")
		source := mockSourceWithComposeFile(t, `
services:
  app:
    build:
      args:
        GREETING: hello
x-topo:
  parameters:
    GREETING: {}
`)

		err := project.Clone(t.Output(), destDir, source, parameter.NewStrictResolverChain(), filepath.Join(destDir, env.DefaultFilename), false)

		require.ErrorIs(t, err, project.ErrParameterMigrationRequired)
		assert.NoDirExists(t, destDir)
	})

	t.Run("configures provided values after migrating legacy build args", func(t *testing.T) {
		destDir := filepath.Join(t.TempDir(), "demo")
		source := mockSourceWithContent(t, map[string]string{
			"compose.yml": `services:
  app:
    build:
      args:
        GREETING: hello
x-topo:
  parameters:
    GREETING: {}
`,
		})
		resolver := parameter.NewStaticResolver(parameter.Values{"GREETING": "configured"})
		var output bytes.Buffer

		err := project.Clone(&output, destDir, source, parameter.NewStrictResolverChain(resolver), filepath.Join(destDir, env.DefaultFilename), true)

		require.NoError(t, err)
		assert.Contains(t, testutil.RequireReadFile(t, filepath.Join(destDir, env.DefaultFilename)), "\nGREETING=\"configured\"\n")
		assert.Contains(t, testutil.RequireReadFile(t, filepath.Join(destDir, "compose.yml")), "GREETING: ${GREETING?configured via topo}")
	})

	t.Run("removes destination directory when migration fails", func(t *testing.T) {
		destDir := filepath.Join(t.TempDir(), "demo")
		source := mockSourceWithContent(t, map[string]string{
			compose.DefaultFileName(): "services: {}\n",
			env.DefaultFilename:       "GREETING=existing\n",
		})
		var output bytes.Buffer

		err := project.Clone(&output, destDir, source, parameter.NewStrictResolverChain(), filepath.Join(destDir, env.DefaultFilename), true)

		require.ErrorContains(t, err, "migration failed: env file already exists")
		assert.NoDirExists(t, destDir)
	})

	t.Run("can configure compose.yml projects", func(t *testing.T) {
		dir := t.TempDir()
		destDir := filepath.Join(dir, "demo")
		mockSource := mockSourceWithContent(t, map[string]string{
			"compose.yml": `
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
`,
		})

		err := project.Clone(t.Output(), destDir, mockSource, parameter.NewStaticResolver(parameter.Values{
			"GREETING": "a-value",
		}), filepath.Join(destDir, env.DefaultFilename), false)

		require.NoError(t, err)
	})
}

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

func mockSourceWithComposeFile(t *testing.T, content string) *mockProjectSource {
	t.Helper()
	return mockSourceWithContent(t, map[string]string{
		compose.DefaultFileName(): content,
	})
}

func mockSourceWithContent(t *testing.T, files map[string]string) *mockProjectSource {
	t.Helper()
	mockSource := &mockProjectSource{}
	mockSource.On("CopyTo", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		destDir := args.String(0)
		testutil.RequireMkdirAll(t, destDir)
		for filename, content := range files {
			testutil.RequireWriteFile(t, filepath.Join(destDir, filename), content)
		}
	})
	t.Cleanup(func() {
		mockSource.AssertExpectations(t)
	})
	return mockSource
}
