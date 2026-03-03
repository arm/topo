package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
)

func TestAnnotateCompatibility(t *testing.T) {
	t.Run("supports template requiring SVE when target has SVE", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "sve-template", Features: []string{"SVE"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.True(t, got[0].Compatibility.Supported)
	})

	t.Run("marks template unsupported when SVE is missing", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "sve-template", Features: []string{"SVE"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.False(t, got[0].Compatibility.Supported)
	})

	t.Run("supports template requiring NEON when target has NEON", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "neon-template", Features: []string{"NEON"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.True(t, got[0].Compatibility.Supported)
	})

	t.Run("marks template unsupported when NEON is missing", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "neon-template", Features: []string{"NEON"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.False(t, got[0].Compatibility.Supported)
	})

	t.Run("marks template unsupported when remoteproc is required but absent", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "rp-template", Features: []string{"remoteproc"}}}
		profile := target.HardwareProfile{}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.False(t, got[0].Compatibility.Supported)
	})

	t.Run("marks template supported when remoteproc is required and present", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "rp-template", Features: []string{"remoteproc"}}}
		profile := target.HardwareProfile{
			RemoteCPU: []target.RemoteprocCPU{
				{Name: "m3"},
			},
		}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.True(t, got[0].Compatibility.Supported)
	})

	t.Run("marks template unsupported when RAM is below requirement", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "ram-template", MinRAMKb: 1024}}
		profile := target.HardwareProfile{TotalMemoryKb: 512}

		got := catalog.AnnotateCompatibility(profile, catalog.WithCompatibility(repos))
		assert.False(t, got[0].Compatibility.Supported)
	})

	t.Run("supports template with no requirements and does not mutate input", func(t *testing.T) {
		repos := []catalog.Repo{
			{Name: "plain"},
		}

		got := catalog.AnnotateCompatibility(target.HardwareProfile{}, catalog.WithCompatibility(repos))
		assert.Len(t, got, 1)
		assert.NotNil(t, got[0].Compatibility)
		assert.True(t, got[0].Compatibility.Supported)
		assert.Equal(t, "plain", repos[0].Name)
	})
}
