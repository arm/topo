package compose_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/arguments"
	"github.com/arm-debug/topo-cli/internal/core/compose"
	"github.com/arm-debug/topo-cli/internal/template"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractNamedServiceVolumes(t *testing.T) {
	t.Run("extracts only named volumes from volume syntax", func(t *testing.T) {
		services := []template.Service{
			{
				Data: map[string]any{
					"volumes": []any{
						"lamb:/var/lib/lamb",
						"/host/path:/container/path",
						"pork:/scratching:ro",
					},
				},
				Name: "a-meaty-service",
			},
			{
				Data: map[string]any{
					"volumes": []any{
						"onion:/var/lib/soup",
						"/host/path:/container/path2",
						"broccoli:/yuck:ro3",
					},
				},
				Name: "a-vegetable-service",
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)
		require.NoError(t, err)
		want := []simpleVolume{
			{Source: "lamb", Target: "/var/lib/lamb"},
			{Source: "pork", Target: "/scratching"},
			{Source: "onion", Target: "/var/lib/soup"},
			{Source: "broccoli", Target: "/yuck"},
		}
		assertVolumesEqual(t, want, volumes)
	})

	t.Run("skips bind mounts", func(t *testing.T) {
		services := []template.Service{
			{
				Data: map[string]any{
					"volumes": []any{
						map[string]any{
							"type":   types.VolumeTypeBind,
							"source": "/host/path",
							"target": "/container/path",
						},
					},
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})

	t.Run("skips tmpfs", func(t *testing.T) {
		services := []template.Service{
			{
				Data: map[string]any{
					"volumes": []any{
						map[string]any{
							"target": "/tmp",
						},
					},
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})

	t.Run("skips volumes with empty source", func(t *testing.T) {
		services := []template.Service{
			{
				Data: map[string]any{
					"volumes": []any{
						map[string]any{
							"source": "",
							"target": "/data",
						},
					},
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes(services)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})
}

func TestCreateService(t *testing.T) {
	t.Run("generates service with extends field", func(t *testing.T) {
		service := template.Service{
			Data: map[string]any{
				"name": "test-service",
				"build": map[string]any{
					"context": ".",
				},
			},
			Name: "test-service-template",
		}

		svc := compose.CreateService("test-service", service, nil)

		assert.Equal(t, "./test-service/compose.yaml", svc.Extends.File)
		assert.Equal(t, "test-service-template", svc.Extends.Service)
	})

	t.Run("injects build arguments", func(t *testing.T) {
		service := template.Service{
			Data: map[string]any{
				"name": "test-service",
				"build": map[string]any{
					"context": ".",
				},
			},
			Name: "test-service-template",
		}
		args := []arguments.ResolvedArg{
			{Name: "GREETING", Value: "Hello"},
			{Name: "PORT", Value: "8080"},
		}

		svc := compose.CreateService("test-service", service, args)

		require.NotNil(t, svc.Build)
		require.NotNil(t, svc.Build.Args)
		assert.Equal(t, "Hello", *svc.Build.Args["GREETING"])
		assert.Equal(t, "8080", *svc.Build.Args["PORT"])
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
