package templates_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintTargetDescription(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("renders host processor details", func(t *testing.T) {
			toPrint := templates.PrintableTargetDescription{
				HardwareProfile: target.HardwareProfile{
					HostProcessor: []target.HostProcessor{
						{Model: "Cortex-A55", Cores: 4, Features: []string{"asimd", "sve"}},
					},
					TotalMemoryKb: 16384,
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Cortex-A55")
			assert.Contains(t, out.String(), "4")
			assert.Contains(t, out.String(), "asimd, sve")
			assert.Contains(t, out.String(), "16384 KB")
		})

		t.Run("renders remote processors when present", func(t *testing.T) {
			toPrint := templates.PrintableTargetDescription{
				HardwareProfile: target.HardwareProfile{
					HostProcessor: []target.HostProcessor{
						{Model: "Cortex-A55", Cores: 2, Features: []string{"asimd"}},
					},
					RemoteCPU: []target.RemoteprocCPU{
						{Name: "remoteproc0"},
						{Name: "remoteproc1"},
					},
					TotalMemoryKb: 8192,
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "Remote Processors")
			assert.Contains(t, out.String(), "remoteproc0")
			assert.Contains(t, out.String(), "remoteproc1")
		})

		t.Run("omits remote processors section when empty", func(t *testing.T) {
			toPrint := templates.PrintableTargetDescription{
				HardwareProfile: target.HardwareProfile{
					HostProcessor: []target.HostProcessor{
						{Model: "Cortex-A55", Cores: 2, Features: []string{"asimd"}},
					},
					TotalMemoryKb: 8192,
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.NotContains(t, out.String(), "Remote Processors")
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders valid JSON with all fields", func(t *testing.T) {
			toPrint := templates.PrintableTargetDescription{
				HardwareProfile: target.HardwareProfile{
					HostProcessor: []target.HostProcessor{
						{Model: "Cortex-A55", Cores: 4, Features: []string{"asimd", "sve"}},
					},
					RemoteCPU: []target.RemoteprocCPU{
						{Name: "remoteproc0"},
					},
					TotalMemoryKb: 16384,
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			assert.Contains(t, out.String(), `"model": "Cortex-A55"`)
			assert.Contains(t, out.String(), `"cores": 4`)
			assert.Contains(t, out.String(), `"remoteproc0"`)
			assert.Contains(t, out.String(), `"totalmemory_kb": 16384`)
		})
	})
}
