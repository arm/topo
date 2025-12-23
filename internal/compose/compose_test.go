package compose_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/compose"
	"github.com/arm-debug/topo-cli/internal/testutil"
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

func TestRead(t *testing.T) {
	t.Run("when project file not found returns error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := compose.Read(dir)

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
		proj, err := compose.Read(composeFilePath)
		require.NoError(t, err)

		got, err := yaml.Marshal(proj)
		require.NoError(t, err)

		assert.YAMLEq(t, composeFileContents, string(got))
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
