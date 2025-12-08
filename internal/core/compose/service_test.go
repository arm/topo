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
		resolved := template.ResolvedTemplate{
			Service: map[string]interface{}{
				"volumes": []interface{}{
					"data:/var/lib/data",
					"/host/path:/container/path",
					"cache:/cache:ro",
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes("test-service", resolved)

		require.NoError(t, err)
		require.Len(t, volumes, 2)
		assert.Equal(t, "data", volumes[0].Source)
		assert.Equal(t, "/var/lib/data", volumes[0].Target)
		assert.Equal(t, "cache", volumes[1].Source)
		assert.Equal(t, "/cache", volumes[1].Target)
	})

	t.Run("skips bind mounts", func(t *testing.T) {
		resolved := template.ResolvedTemplate{
			Service: map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"type":   types.VolumeTypeBind,
						"source": "/host/path",
						"target": "/container/path",
					},
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes("test-service", resolved)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})

	t.Run("skips tmpfs", func(t *testing.T) {
		resolved := template.ResolvedTemplate{
			Service: map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"target": "/tmp",
					},
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes("test-service", resolved)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})

	t.Run("skips volumes with empty source", func(t *testing.T) {
		resolved := template.ResolvedTemplate{
			Service: map[string]interface{}{
				"volumes": []interface{}{
					map[string]interface{}{
						"source": "",
						"target": "/data",
					},
				},
			},
		}

		volumes, err := compose.ExtractNamedServiceVolumes("test-service", resolved)

		require.NoError(t, err)
		assert.Empty(t, volumes)
	})
}

func TestCreateService(t *testing.T) {
	t.Run("generates service with extends field", func(t *testing.T) {
		resolved := template.ResolvedTemplate{
			Service: map[string]interface{}{
				"name": "test-service",
				"build": map[string]interface{}{
					"context": ".",
				},
			},
			ServiceName: "test-service-template",
			Args:        nil,
		}

		svc := compose.CreateService("test-service", resolved)

		assert.Equal(t, "./test-service/compose.yaml", svc.Extends.File)
		assert.Equal(t, "test-service-template", svc.Extends.Service)
	})

	t.Run("injects build arguments", func(t *testing.T) {
		resolved := template.ResolvedTemplate{
			Service: map[string]interface{}{
				"name": "test-service",
				"build": map[string]interface{}{
					"context": ".",
				},
			},
			ServiceName: "test-service-template",
			Args: []arguments.ResolvedArg{
				{Name: "GREETING", Value: "Hello"},
				{Name: "PORT", Value: "8080"},
			},
		}

		svc := compose.CreateService("test-service", resolved)

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
