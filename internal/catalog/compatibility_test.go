package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/probe"
	"github.com/stretchr/testify/assert"
)

func TestAnnotateCompatibility(t *testing.T) {
	t.Run("supports template requiring SVE when target has SVE", func(t *testing.T) {
		repo := catalog.Repo{Name: "sve-template", Features: []string{"SVE"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when SVE is missing", func(t *testing.T) {
		repo := catalog.Repo{Name: "sve-template", Features: []string{"SVE"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports template requiring NEON when target has NEON", func(t *testing.T) {
		repo := catalog.Repo{Name: "neon-template", Features: []string{"NEON"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when NEON is missing", func(t *testing.T) {
		repo := catalog.Repo{Name: "neon-template", Features: []string{"NEON"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when remoteproc is required but absent", func(t *testing.T) {
		repo := catalog.Repo{Name: "rp-template", Features: []string{"remoteproc"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template supported when remoteproc is required and present", func(t *testing.T) {
		repo := catalog.Repo{Name: "rp-template", Features: []string{"remoteproc"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			RemoteProcessors: []probe.RemoteProcessor{
				{Name: "m3"},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports template when any required feature is present", func(t *testing.T) {
		repo := catalog.Repo{Name: "multi-feature-template", Features: []string{"SVE", "remoteproc"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when none of required features are present", func(t *testing.T) {
		repo := catalog.Repo{Name: "multi-feature-template", Features: []string{"SVE", "remoteproc"}}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when RAM is below requirement", func(t *testing.T) {
		repo := catalog.Repo{Name: "ram-template", MinRAMKb: 1024}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{TotalMemoryKb: 512}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports template with no requirements and does not mutate input", func(t *testing.T) {
		repo := catalog.Repo{Name: "plain"}
		repos := []catalog.Repo{repo}
		profile := probe.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("leaves compatibility unknown when profile is nil", func(t *testing.T) {
		repo := catalog.Repo{Name: "plain"}
		repos := []catalog.Repo{repo}

		got := catalog.AnnotateCompatibility(nil, repos)
		want := []catalog.RepoWithCompatibility{
			{Repo: repo, Compatibility: catalog.CompatibilityUnknown},
		}
		assert.Equal(t, want, got)
	})
}
