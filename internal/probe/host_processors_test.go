package probe_test

import (
	"context"
	"testing"

	"github.com/arm/topo/internal/probe"
	"github.com/arm/topo/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func lscpuJSON(fields string) string {
	return `{"lscpu": [` + fields + `]}`
}

func TestProbeCPU(t *testing.T) {
	t.Run("parses lscpu with clusters", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				"lscpu --json": {Output: lscpuJSON(`
					{"field": "Vendor ID:", "data": "ARM"},
					{"field": "Model name:", "data": "Cortex-A55"},
					{"field": "Core(s) per cluster:", "data": "2"},
					{"field": "Socket(s):", "data": "-"},
					{"field": "Cluster(s):", "data": "1"},
					{"field": "Flags:", "data": "fp asimd"}
				`)},
			},
		}

		got, err := probe.HostProcessors(context.Background(), r)

		require.NoError(t, err)
		want := []probe.HostProcessor{
			{
				Model:    "Cortex-A55",
				Cores:    2,
				Features: []string{"fp", "asimd"},
			},
		}
		assert.Equal(t, want, got)
	})

	t.Run("parses lscpu with sockets", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				"lscpu --json": {Output: lscpuJSON(`
					{"field": "Vendor ID:", "data": "ARM"},
					{"field": "Model name:", "data": "Cortex-A72"},
					{"field": "Core(s) per socket:", "data": "4"},
					{"field": "Socket(s):", "data": "2"},
					{"field": "Flags:", "data": "fp asimd evtstrm"}
				`)},
			},
		}

		got, err := probe.HostProcessors(context.Background(), r)

		require.NoError(t, err)
		require.Len(t, got, 1)
		want := probe.HostProcessor{
			Model:    "Cortex-A72",
			Cores:    8,
			Features: []string{"fp", "asimd", "evtstrm"},
		}
		assert.Equal(t, want, got[0])
	})

	t.Run("parses multiple processors", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				"lscpu --json": {Output: lscpuJSON(`
					{"field": "Vendor ID:", "data": "ARM"},
					{"field": "Model name:", "data": "Cortex-A55"},
					{"field": "Core(s) per socket:", "data": "4"},
					{"field": "Socket(s):", "data": "1"},
					{"field": "Flags:", "data": "fp asimd"},
					{"field": "Model name:", "data": "Cortex-A78"},
					{"field": "Core(s) per socket:", "data": "2"},
					{"field": "Socket(s):", "data": "1"},
					{"field": "Flags:", "data": "fp asimd sve"}
				`)},
			},
		}

		got, err := probe.HostProcessors(context.Background(), r)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "Cortex-A55", got[0].Model)
		assert.Equal(t, 4, got[0].Cores)
		assert.Equal(t, "Cortex-A78", got[1].Model)
		assert.Equal(t, 2, got[1].Cores)
	})

	t.Run("returns empty when no model name field", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				"lscpu --json": {Output: lscpuJSON(`
					{"field": "Architecture:", "data": "aarch64"}
				`)},
			},
		}

		got, err := probe.HostProcessors(context.Background(), r)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error when cores per socket is not a number", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				"lscpu --json": {Output: lscpuJSON(`
					{"field": "Model name:", "data": "Cortex-A55"},
					{"field": "Core(s) per socket:", "data": "abc"},
					{"field": "Socket(s):", "data": "1"}
				`)},
			},
		}

		_, err := probe.HostProcessors(context.Background(), r)

		assert.Error(t, err)
	})

	t.Run("returns error when lscpu not found", func(t *testing.T) {
		r := &runner.Fake{}

		_, err := probe.HostProcessors(context.Background(), r)

		assert.ErrorContains(t, err, `"lscpu" not found in $PATH`)
	})

	t.Run("returns error when lscpu output is invalid JSON", func(t *testing.T) {
		r := &runner.Fake{
			Binaries: []string{"lscpu"},
			Commands: map[string]runner.FakeResult{
				"lscpu --json": {Output: "not json"},
			},
		}

		_, err := probe.HostProcessors(context.Background(), r)

		assert.Error(t, err)
	})
}

func TestHostProcessor(t *testing.T) {
	t.Run("ExtractArmFeatures", func(t *testing.T) {
		t.Run("extracts mapped Arm features and ignores unrecognised", func(t *testing.T) {
			ts := probe.HostProcessor{
				Features: []string{"fp", "asimd", "sve2", "sme"},
			}

			res := ts.ExtractArmFeatures()

			want := []string{"NEON", "SVE2", "SME"}
			assert.Equal(t, want, res)
		})

		t.Run("returns empty slice if no matching features", func(t *testing.T) {
			ts := probe.HostProcessor{
				Features: []string{"fp", "crc32"},
			}

			res := ts.ExtractArmFeatures()

			assert.Empty(t, res)
		})
	})
}
