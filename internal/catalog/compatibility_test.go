package catalog_test

import (
	"testing"

	"github.com/arm/topo/internal/catalog"
	"github.com/arm/topo/internal/probe"
	"github.com/stretchr/testify/assert"
)

func TestAnnotateCompatibility(t *testing.T) {
	t.Run("supports template requiring SVE when target has SVE", func(t *testing.T) {
		template := catalog.Template{Name: "sve-template", Features: []string{"SVE"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when SVE is missing", func(t *testing.T) {
		template := catalog.Template{Name: "sve-template", Features: []string{"SVE"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports template requiring NEON when target has NEON", func(t *testing.T) {
		template := catalog.Template{Name: "neon-template", Features: []string{"NEON"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when NEON is missing", func(t *testing.T) {
		template := catalog.Template{Name: "neon-template", Features: []string{"NEON"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when remoteproc is required but absent", func(t *testing.T) {
		template := catalog.Template{Name: "rp-template", Features: []string{"remoteproc"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template supported when remoteproc is required and present", func(t *testing.T) {
		template := catalog.Template{Name: "rp-template", Features: []string{"remoteproc"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			RemoteProcessors: []probe.RemoteProcessor{
				{Name: "m3"},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports template when any required feature is present", func(t *testing.T) {
		template := catalog.Template{Name: "multi-feature-template", Features: []string{"SVE", "remoteproc"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd", "sve"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when none of required features are present", func(t *testing.T) {
		template := catalog.Template{Name: "multi-feature-template", Features: []string{"SVE", "remoteproc"}}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{
			HostProcessors: []probe.HostProcessor{
				{Features: []string{"asimd"}},
			},
		}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("marks template unsupported when RAM is below requirement", func(t *testing.T) {
		template := catalog.Template{Name: "ram-template", MinRAMKb: 1024}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{TotalMemoryKb: 512}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilityUnsupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("supports template with no requirements and does not mutate input", func(t *testing.T) {
		template := catalog.Template{Name: "plain"}
		templates := []catalog.Template{template}
		profile := probe.HardwareProfile{}

		got := catalog.AnnotateCompatibility(&profile, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilitySupported},
		}
		assert.Equal(t, want, got)
	})

	t.Run("leaves compatibility unknown when profile is nil", func(t *testing.T) {
		template := catalog.Template{Name: "plain"}
		templates := []catalog.Template{template}

		got := catalog.AnnotateCompatibility(nil, templates)
		want := []catalog.TemplateWithCompatibility{
			{Template: template, Compatibility: catalog.CompatibilityUnknown},
		}
		assert.Equal(t, want, got)
	})
}
