package views_test

import (
	"bytes"
	"testing"

	"github.com/arm/topo/internal/output/term"
	"github.com/arm/topo/internal/output/views"
	"github.com/arm/topo/internal/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetDescription(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("outputs valid yaml that round-trips back to the hardware profile", func(t *testing.T) {
			profile := probe.HardwareProfile{
				HostProcessors: []probe.HostProcessor{
					{Model: "Cortex-A55", Cores: 4, Features: []string{"asimd", "sve"}},
				},
				RemoteProcessors: []probe.RemoteProcessor{
					{Name: "remoteproc0"},
				},
				TotalMemoryKb: 16384,
			}
			toPrint := views.TargetDescription{HardwareProfile: profile}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			want := `
hostProcessors:
  - model: Cortex-A55
    cores: 4
    features:
      - asimd
      - sve
remoteProcessors:
  - name: remoteproc0
totalMemoryKb: 16384
`
			require.NoError(t, err)
			assert.YAMLEq(t, want, out.String())
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders valid JSON with all fields", func(t *testing.T) {
			toPrint := views.TargetDescription{
				HardwareProfile: probe.HardwareProfile{
					HostProcessors: []probe.HostProcessor{
						{Model: "Cortex-A55", Cores: 4, Features: []string{"asimd", "sve"}},
					},
					RemoteProcessors: []probe.RemoteProcessor{
						{Name: "remoteproc0"},
					},
					TotalMemoryKb: 16384,
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			want := `{
				"hostProcessors": [
					{
						"model": "Cortex-A55",
						"cores": 4,
						"features": ["asimd", "sve"]
					}
				],
				"remoteProcessors": [
					{"name": "remoteproc0"}
				],
				"totalMemoryKb": 16384
			}`
			require.NoError(t, err)
			assert.JSONEq(t, want, out.String())
		})
	})
}
