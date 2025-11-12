package compose_test

import (
	"testing"

	"github.com/arm-debug/topo-cli/internal/core/compose"
	"github.com/arm-debug/topo-cli/internal/service"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServiceFromTopo(t *testing.T) {
	t.Run("sets name and build context", func(t *testing.T) {
		topoService := service.TemplateManifest{
			Name:        "test",
			Description: "A test service",
			Service: map[string]interface{}{
				"runtime": "cool-topo-runtime",
			},
		}

		svc, err := compose.ParseServiceFromTopo("test-service", &topoService)
		require.NoError(t, err)

		assert.Equal(t, "test-service", svc.Name)
		require.NotNil(t, svc.Build)
		assert.Equal(t, "./test-service", svc.Build.Context)
		assert.Equal(t, "cool-topo-runtime", svc.Runtime)
	})

	t.Run("handles short volume syntax", func(t *testing.T) {
		topoService := service.TemplateManifest{
			Service: map[string]interface{}{
				"volumes": []interface{}{"data:/var/lib/data"},
			},
		}

		svc, err := compose.ParseServiceFromTopo("test-service", &topoService)
		require.NoError(t, err)

		require.Len(t, svc.Volumes, 1)
		assert.Equal(t, "data", svc.Volumes[0].Source)
		assert.Equal(t, "/var/lib/data", svc.Volumes[0].Target)
	})
}

func TestRegisterNamedVolumes(t *testing.T) {
	t.Run("registers named volume", func(t *testing.T) {
		project := &types.Project{
			Volumes: nil,
		}
		svc := types.ServiceConfig{
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "mydata", Target: "/data"},
			},
		}

		compose.RegisterNamedVolumes(project, svc)

		assert.Equal(t, types.Volumes{"mydata": {}}, project.Volumes)
	})

	t.Run("skips bind mounts", func(t *testing.T) {
		project := &types.Project{
			Volumes: nil,
		}
		svc := types.ServiceConfig{
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeBind, Source: "/host/path", Target: "/container/path"},
			},
		}

		compose.RegisterNamedVolumes(project, svc)

		assert.Empty(t, project.Volumes)
	})

	t.Run("skips tmpfs", func(t *testing.T) {
		project := &types.Project{
			Volumes: nil,
		}
		svc := types.ServiceConfig{
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeTmpfs, Target: "/tmp"},
			},
		}

		compose.RegisterNamedVolumes(project, svc)

		assert.Empty(t, project.Volumes)
	})

	t.Run("skips volumes with empty source", func(t *testing.T) {
		project := &types.Project{
			Volumes: nil,
		}
		svc := types.ServiceConfig{
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "", Target: "/data"},
			},
		}

		compose.RegisterNamedVolumes(project, svc)

		assert.Empty(t, project.Volumes)
	})

	t.Run("does not overwrite existing volumes", func(t *testing.T) {
		project := &types.Project{
			Volumes: types.Volumes{
				"existing": types.VolumeConfig{Name: "existing", Driver: "local"},
			},
		}
		svc := types.ServiceConfig{
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "existing", Target: "/data"},
				{Type: types.VolumeTypeVolume, Source: "new", Target: "/other"},
			},
		}

		compose.RegisterNamedVolumes(project, svc)

		assert.Equal(t, types.Volumes{
			"existing": types.VolumeConfig{Name: "existing", Driver: "local"},
			"new":      types.VolumeConfig{},
		}, project.Volumes)
	})

	t.Run("handles multiple named volumes", func(t *testing.T) {
		project := &types.Project{
			Volumes: nil,
		}
		svc := types.ServiceConfig{
			Volumes: []types.ServiceVolumeConfig{
				{Type: types.VolumeTypeVolume, Source: "data", Target: "/data"},
				{Type: types.VolumeTypeVolume, Source: "cache", Target: "/cache"},
				{Type: types.VolumeTypeBind, Source: "/host", Target: "/mnt"},
			},
		}

		compose.RegisterNamedVolumes(project, svc)

		assert.Equal(t, types.Volumes{
			"data":  types.VolumeConfig{},
			"cache": types.VolumeConfig{},
		}, project.Volumes)
	})
}
