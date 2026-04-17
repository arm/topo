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
		t.Run("outputs valid yaml that round-trips back to the hardware profile", func(t *testing.T) {
			profile := target.HardwareProfile{
				HostProcessor: []target.HostProcessor{
					{Model: "Cortex-A55", Cores: 4, Features: []string{"asimd", "sve"}},
				},
				RemoteCPU: []target.RemoteprocCPU{
					{Name: "remoteproc0"},
				},
				TotalMemoryKb: 16384,
			}
			toPrint := templates.PrintableTargetDescription{HardwareProfile: profile}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			want := `
host:
  - model: Cortex-A55
    cores: 4
    features:
      - asimd
      - sve
remoteprocs:
  - name: remoteproc0
totalmemory_kb: 16384
`
			require.NoError(t, err)
			assert.YAMLEq(t, want, out.String())
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

			want := `{
				"host": [
					{
						"model": "Cortex-A55",
						"cores": 4,
						"features": ["asimd", "sve"]
					}
				],
				"remoteprocs": [
					{"name": "remoteproc0"}
				],
				"totalmemory_kb": 16384
			}`
			require.NoError(t, err)
			assert.JSONEq(t, want, out.String())
		})
	})
}
