package compose_test

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/arm/topo/internal/compose"
	"github.com/arm/topo/internal/testutil"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExtractNamedServiceVolumes(t *testing.T) {
	t.Run("extracts only named volumes from volume syntax", func(t *testing.T) {
		services := map[string]any{
			"volumes": []any{
				"lamb:/var/lib/lamb",
				"/host/path:/container/path",
				"pork:/scratching:ro",
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)
		require.NoError(t, err)
		want := []simpleVolume{
			{Source: "lamb", Target: "/var/lib/lamb"},
			{Source: "pork", Target: "/scratching"},
		}
		assertVolumesEqual(t, want, volumes)
	})

	t.Run("skips bind mounts", func(t *testing.T) {
		services := map[string]any{
			"volumes": []any{
				map[string]any{
					"type":   types.VolumeTypeBind,
					"source": "/host/path",
					"target": "/container/path",
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})

	t.Run("skips tmpfs", func(t *testing.T) {
		services := map[string]any{
			"volumes": []any{
				map[string]any{
					"target": "/tmp",
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})

	t.Run("skips volumes with empty source", func(t *testing.T) {
		services := map[string]any{
			"volumes": []any{
				map[string]any{
					"source": "",
					"target": "/data",
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})
}

func TestCreateServiceByExtension(t *testing.T) {
	t.Run("generates service with extends field", func(t *testing.T) {
		serviceName := "very-fancy-service"
		composeFilePath := "a-path-to-service/compose.yaml"

		got := compose.CreateServiceByExtension(composeFilePath, serviceName, nil)

		assert.Equal(t, composeFilePath, got.Extends.File)
		assert.Equal(t, serviceName, got.Extends.Service)
	})

	t.Run("injects build arguments", func(t *testing.T) {
		args := map[string]string{
			"GREETING": "Hello",
			"PORT":     "8080",
		}

		got := compose.CreateServiceByExtension("some-path.yaml", "some-name", args)

		require.NotNil(t, got.Build)
		require.NotNil(t, got.Build.Args)
		assert.Equal(t, "Hello", *got.Build.Args["GREETING"])
		assert.Equal(t, "8080", *got.Build.Args["PORT"])
	})
}

func TestFilterResolvedBuildArgs(t *testing.T) {
	t.Run("returns only args referenced by build args mapping", func(t *testing.T) {
		service := map[string]any{
			"build": map[string]any{
				"args": map[string]any{
					"GREETING": "${GREETING:-hello}",
				},
			},
		}

		resolved := map[string]string{
			"GREETING": "Hello",
			"PORT":     "8080",
		}

		got := compose.FilterResolvedBuildArgs(service, resolved)

		assert.Equal(t, map[string]string{"GREETING": "Hello"}, got)
	})

	t.Run("returns only args referenced by build args sequence", func(t *testing.T) {
		service := map[string]any{
			"build": map[string]any{
				"args": []any{"GREETING", "PORT=80"},
			},
		}

		resolved := map[string]string{
			"GREETING": "Hello",
			"PORT":     "8080",
			"UNUSED":   "value",
		}

		got := compose.FilterResolvedBuildArgs(service, resolved)

		assert.Equal(t, map[string]string{"GREETING": "Hello", "PORT": "8080"}, got)
	})

	t.Run("returns empty when service has no build args", func(t *testing.T) {
		service := map[string]any{"image": "nginx:alpine"}
		resolved := map[string]string{"GREETING": "Hello"}

		got := compose.FilterResolvedBuildArgs(service, resolved)

		assert.Empty(t, got)
	})
}

func TestRegisterVolumes(t *testing.T) {
	t.Run("registers volumes", func(t *testing.T) {
		project := &types.Project{
			Volumes: nil,
		}
		volumes := []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeVolume, Source: "mydata", Target: "/data"},
		}

		compose.RegisterVolumes(project, volumes)

		assert.Equal(t, types.Volumes{"mydata": {}}, project.Volumes)
	})

	t.Run("does not overwrite existing volumes", func(t *testing.T) {
		project := &types.Project{
			Volumes: types.Volumes{
				"existing": types.VolumeConfig{Name: "existing", Driver: "local"},
			},
		}
		volumes := []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeVolume, Source: "existing", Target: "/data"},
			{Type: types.VolumeTypeVolume, Source: "new", Target: "/other"},
		}

		compose.RegisterVolumes(project, volumes)

		assert.Equal(t, types.Volumes{
			"existing": types.VolumeConfig{Name: "existing", Driver: "local"},
			"new":      types.VolumeConfig{},
		}, project.Volumes)
	})
}

func TestImageNames(t *testing.T) {
	t.Run("returns explicit and generated image names in sorted order", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `
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

		got, err := compose.ImageNames(path)

		require.NoError(t, err)
		assert.Equal(t, []string{"nginx:1.27", "springfield-api", "worker:dev"}, got)
	})

	t.Run("uses the compose file directory when project name is omitted", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "compose.yaml")
		testutil.RequireWriteFile(t, path, `
services:
  api:
    build: .
  web:
    image: nginx:1.27
`)

		got, err := compose.ImageNames(path)

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{filepath.Base(dir) + "-api", "nginx:1.27"}, got)
	})

	t.Run("resolves image names from extended services", func(t *testing.T) {
		dir := t.TempDir()
		basePath := filepath.Join(dir, "base.yaml")
		path := filepath.Join(dir, "compose.yaml")
		testutil.RequireWriteFile(t, basePath, `
services:
  image-base:
    image: duff:latest
  build-base:
    build: .
`)
		testutil.RequireWriteFile(t, path, `
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

		got, err := compose.ImageNames(path)

		require.NoError(t, err)
		assert.Equal(t, []string{"duff:latest", "springfield-api"}, got)
	})

	t.Run("returns sorted output", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `
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

		got, err := compose.ImageNames(path)

		require.NoError(t, err)
		assert.True(t, sort.StringsAreSorted(got))
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `{invalid`)

		_, err := compose.ImageNames(path)

		assert.Error(t, err)
	})
}

func TestPullableServices(t *testing.T) {
	t.Run("returns services without a build key", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `
services:
  duff-beer:
    image: duff:7
  kwik-e-mart:
    image: apu:16
`)

		got, err := compose.PullableServices(path)

		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"duff-beer", "kwik-e-mart"}, got)
	})

	t.Run("excludes services with a build key", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `
services:
  krusty-burger:
    build: .
    image: krusty:latest
  duff-beer:
    image: duff:7
`)

		got, err := compose.PullableServices(path)

		require.NoError(t, err)
		assert.Equal(t, []string{"duff-beer"}, got)
	})

	t.Run("returns empty slice when all services are buildable", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `
services:
  krusty-burger:
    build: .
  nuclear-plant:
    build:
      context: .
      dockerfile: Dockerfile.sector7g
`)

		got, err := compose.PullableServices(path)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("excludes services that extend a buildable service", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `
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

		got, err := compose.PullableServices(path)

		require.NoError(t, err)
		assert.Equal(t, []string{"duff-beer"}, got)
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "compose.yaml")
		testutil.RequireWriteFile(t, path, `{invalid`)

		_, err := compose.PullableServices(path)

		assert.Error(t, err)
	})
}

func TestReadProject(t *testing.T) {
	t.Run("when project file not found returns error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := compose.ReadProject(dir)

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
		composeFilePath := testutil.WriteComposeFile(t, dir, composeFileContents)
		proj, err := compose.ReadProject(composeFilePath)
		require.NoError(t, err)

		got, err := yaml.Marshal(proj)
		require.NoError(t, err)

		assert.YAMLEq(t, composeFileContents, string(got))
	})
}

func TestWriteProject(t *testing.T) {
	t.Run("writes project to compose file", func(t *testing.T) {
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
		temporaryComposeFilePath := filepath.Join(t.TempDir(), "expected-compose.yml")
		testutil.RequireWriteFile(t, temporaryComposeFilePath, composeFileContents)
		proj, err := compose.ReadProject(temporaryComposeFilePath)
		require.NoError(t, err)
		composeFilePath := filepath.Join(t.TempDir(), "test-compose.yml")

		err = compose.WriteProject(proj, composeFilePath)
		require.NoError(t, err)

		got := testutil.RequireReadFile(t, composeFilePath)
		assert.YAMLEq(t, composeFileContents, got)
	})
}

type simpleVolume struct {
	Source string
	Target string
}

func assertVolumesEqual(t *testing.T, want []simpleVolume, got []types.ServiceVolumeConfig) {
	var gotSimpleVolumes []simpleVolume
	for _, v := range got {
		gotSimpleVolumes = append(gotSimpleVolumes, simpleVolume{Source: v.Source, Target: v.Target})
	}
	assert.ElementsMatch(t, want, gotSimpleVolumes)
}
