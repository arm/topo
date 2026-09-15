package project_test

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestImageNames(t *testing.T) {
	t.Run("returns explicit and generated image names", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `
name: springfield
services:
  api:
    build: .
  web:
    image: nginx:1.27
  worker:
    build: .
    image: worker:dev
`)

		got, err := project.ImageNames(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"nginx:1.27", "springfield-api", "worker:dev"}, got)
	})

	t.Run("uses the compose file directory when project name is omitted", func(t *testing.T) {
		dir := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, dir, `
services:
  api:
    build: .
  web:
    image: nginx:1.27
`)

		got, err := project.ImageNames(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{filepath.Base(dir) + "-api", "nginx:1.27"}, got)
	})

	t.Run("resolves image names from extended services", func(t *testing.T) {
		dir := t.TempDir()
		basePath := filepath.Join(dir, "base.yaml")
		testutil.RequireWriteFile(t, basePath, `
services:
  image-base:
    image: duff:latest
  build-base:
    build: .
`)
		path := testutil.RequireWriteComposeFile(t, dir, `
name: springfield
services:
  duff:
    extends:
      file: base.yaml
      service: image-base
  api:
    extends:
      file: base.yaml
      service: build-base
`)

		got, err := project.ImageNames(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"duff:latest", "springfield-api"}, got)
	})

	t.Run("returns sorted output", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `
name: springfield
services:
  zulu:
    build: .
  alpha:
    image: alpine:3.20
  mike:
    build: .
    image: mike:dev
  beta:
    image: busybox:1.36
`)

		got, err := project.ImageNames(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.True(t, sort.StringsAreSorted(got))
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `{invalid`)

		_, err := project.ImageNames(project.Scope{ComposeFile: path})

		assert.Error(t, err)
	})
}

func TestPullableServices(t *testing.T) {
	t.Run("returns services without a build key", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  duff-beer:
    image: duff:7
  kwik-e-mart:
    image: apu:16
`)

		got, err := project.PullableServices(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"duff-beer", "kwik-e-mart"}, got)
	})

	t.Run("excludes services with a build key", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  krusty-burger:
    build: .
    image: krusty:latest
  duff-beer:
    image: duff:7
`)

		got, err := project.PullableServices(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Equal(t, []string{"duff-beer"}, got)
	})

	t.Run("returns empty slice when all services are buildable", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  krusty-burger:
    build: .
  nuclear-plant:
    build:
      context: .
      dockerfile: Dockerfile.sector7g
`)

		got, err := project.PullableServices(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("excludes services that extend a buildable service", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `
services:
  krusty-burger:
    build: .
    image: krusty:latest
  ribwich:
    extends:
      service: krusty-burger
  duff-beer:
    image: duff:7
`)

		got, err := project.PullableServices(project.Scope{ComposeFile: path})

		require.NoError(t, err)
		assert.Equal(t, []string{"duff-beer"}, got)
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), `{invalid`)

		_, err := project.PullableServices(project.Scope{ComposeFile: path})

		assert.Error(t, err)
	})
}

func TestRead(t *testing.T) {
	t.Run("when project file not found returns error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := project.Read(project.Scope{ComposeFile: filepath.Join(dir, "compose.yaml")})

		assert.Error(t, err)
	})

	t.Run("when project file found returns correct project type", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := `
name: test
services:
  test-service:
    build:
      context: .
      args:
        FOO: new-foo
        BAR: new-bar
`
		composeFilePath := testutil.RequireWriteComposeFile(t, dir, composeFileContents)
		composeProject, err := project.Read(project.Scope{ComposeFile: composeFilePath})
		require.NoError(t, err)

		got, err := yaml.Marshal(composeProject)
		require.NoError(t, err)

		assert.YAMLEq(t, composeFileContents, string(got))
	})

	t.Run("inherits environment variables from custom dotenv file", func(t *testing.T) {
		dir := t.TempDir()
		serviceName := "test-service"
		composeFileContents := fmt.Sprintf(`
name: test
services:
  %s:
    image: ${IMAGE_NAME}
`, serviceName)
		composeFilePath := testutil.RequireWriteComposeFile(t, dir, composeFileContents)
		imageName := "image-from-env"
		envFile := filepath.Join(dir, ".something.env")
		testutil.RequireWriteFile(t, envFile, fmt.Sprintf("IMAGE_NAME=%s", imageName))

		composeProject, err := project.Read(project.Scope{ComposeFile: composeFilePath, EnvFiles: []string{envFile}})
		require.NoError(t, err)

		require.Contains(t, composeProject.Services, serviceName)
		require.Equal(t, composeProject.Services[serviceName].Image, imageName)
	})

	t.Run("inherits env vars from environment", func(t *testing.T) {
		dir := t.TempDir()
		serviceName := "test-service"
		composeFileContents := fmt.Sprintf(`
name: test
services:
  %s:
    image: ${IMAGE_NAME}
`, serviceName)
		composeFilePath := testutil.RequireWriteComposeFile(t, dir, composeFileContents)
		imageName := "image-from-env"
		t.Setenv("IMAGE_NAME", imageName)

		composeProject, err := project.Read(project.Scope{ComposeFile: composeFilePath})
		require.NoError(t, err)

		require.Contains(t, composeProject.Services, serviceName)
		require.Equal(t, composeProject.Services[serviceName].Image, imageName)
	})

	t.Run("inherits env vars from project scope", func(t *testing.T) {
		dir := t.TempDir()
		composeFileContents := "name: ${NAME:?project name is required}"
		composeFilePath := testutil.RequireWriteComposeFile(t, dir, composeFileContents)
		projectName := "project-from-env"
		scope := project.Scope{ComposeFile: composeFilePath, Env: []string{fmt.Sprintf("%s=%s", "NAME", projectName)}}

		composeProject, err := project.Read(scope)
		require.NoError(t, err)

		require.Equal(t, composeProject.Name, projectName)
	})
}
