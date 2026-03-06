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

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilitySupported, got[0].Compatibility)
	})

	t.Run("marks template unsupported when SVE is missing", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "sve-template", Features: []string{"SVE"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilityUnsupported, got[0].Compatibility)
	})

	t.Run("supports template requiring NEON when target has NEON", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "neon-template", Features: []string{"NEON"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilitySupported, got[0].Compatibility)
	})

	t.Run("marks template unsupported when NEON is missing", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "neon-template", Features: []string{"NEON"}}}
		profile := target.HardwareProfile{
			HostProcessor: []target.HostProcessor{
				{Features: []string{"sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilityUnsupported, got[0].Compatibility)
	})

	t.Run("marks template unsupported when remoteproc is required but absent", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "rp-template", Features: []string{"remoteproc"}}}
		profile := target.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilityUnsupported, got[0].Compatibility)
	})

	t.Run("marks template supported when remoteproc is required and present", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "rp-template", Features: []string{"remoteproc"}}}
		profile := target.HardwareProfile{
			RemoteCPU: []target.RemoteprocCPU{
				{Name: "m3"},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilitySupported, got[0].Compatibility)
	})

	t.Run("marks template unsupported when RAM is below requirement", func(t *testing.T) {
		repos := []catalog.Repo{{Name: "ram-template", MinRAMKb: 1024}}
		profile := target.HardwareProfile{TotalMemoryKb: 512}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Equal(t, catalog.CompatibilityUnsupported, got[0].Compatibility)
	})

	t.Run("supports template with no requirements and does not mutate input", func(t *testing.T) {
		repos := []catalog.Repo{
			{Name: "plain"},
		}
		profile := target.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, repos)
		assert.Len(t, got, 1)
		assert.Equal(t, catalog.CompatibilitySupported, got[0].Compatibility)
		assert.Equal(t, "plain", repos[0].Name)
	})

	t.Run("leaves compatibility unknown when profile is nil", func(t *testing.T) {
		repos := []catalog.Repo{
			{Name: "plain"},
		}

		got := catalog.AnnotateCompatibility(nil, repos)
		assert.Len(t, got, 1)
		assert.Equal(t, catalog.CompatibilityUnknown, got[0].Compatibility)
	})
}
