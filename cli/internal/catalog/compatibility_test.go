package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/probe"
	"github.com/stretchr/testify/assert"
)

func TestAnnotateCompatibility(t *testing.T) {
	t.Run("supports project requiring SVE when target has SVE", func(t *testing.T) {
		project := catalog.Project{Name: "sve-project", Features: []string{"SVE"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks project unsupported when SVE is missing", func(t *testing.T) {
		project := catalog.Project{Name: "sve-project", Features: []string{"SVE"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports project requiring NEON when target has NEON", func(t *testing.T) {
		project := catalog.Project{Name: "neon-project", Features: []string{"NEON"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks project unsupported when NEON is missing", func(t *testing.T) {
		project := catalog.Project{Name: "neon-project", Features: []string{"NEON"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks project unsupported when remoteproc is required but absent", func(t *testing.T) {
		project := catalog.Project{Name: "rp-project", Features: []string{"remoteproc"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks project supported when remoteproc is required and present", func(t *testing.T) {
		project := catalog.Project{Name: "rp-project", Features: []string{"remoteproc"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			RemoteProcessors: []probe.RemoteProcessor{
				{Name: "m3"},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports project when any required feature is present", func(t *testing.T) {
		project := catalog.Project{Name: "multi-feature-project", Features: []string{"SVE", "remoteproc"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks project unsupported when none of required features are present", func(t *testing.T) {
		project := catalog.Project{Name: "multi-feature-project", Features: []string{"SVE", "remoteproc"}}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports project with no requirements and does not mutate input", func(t *testing.T) {
		project := catalog.Project{Name: "plain"}
		projects := []catalog.Project{project}
		profile := probe.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("leaves compatibility unknown when profile is nil", func(t *testing.T) {
		project := catalog.Project{Name: "plain"}
		projects := []catalog.Project{project}

		got := catalog.AnnotateCompatibility(nil, projects)
		want := []catalog.ProjectWithCompatibility{
			{Project: project, Compatibility: catalog.CompatibilityUnknown},
		}
		assert.Equal(t, want, got)
	})
}
