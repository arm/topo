package templates_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/arm/topo/internal/output/printable"
	"github.com/arm/topo/internal/output/templates"
	"github.com/arm/topo/internal/output/term"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintPSReport(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("renders container image, status, and ports", func(t *testing.T) {
			toPrint := templates.PrintablePSReport{
				Containers: []templates.ContainerStatus{
					{
						Image:  "my-app",
						Status: "Up 5 minutes",
						Ports:  "localhost:8080",
					},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "my-app")
			assert.Contains(t, out.String(), "Up 5 minutes")
			assert.Contains(t, out.String(), "localhost:8080")
		})

		t.Run("renders multiple containers", func(t *testing.T) {
			toPrint := templates.PrintablePSReport{
				Containers: []templates.ContainerStatus{
					{Image: "web"},
					{Image: "db"},
				},
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "web")
			assert.Contains(t, out.String(), "db")
		})

		t.Run("renders empty message when no containers", func(t *testing.T) {
			toPrint := templates.PrintablePSReport{Containers: nil}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "No containers deployed from this project are running.")
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
			toPrint := templates.PrintablePSReport{
				Containers: []templates.ContainerStatus{
					{
						Image:  "my-app",
						Status: "Up 5 minutes",
						Ports:  "localhost:8080",
					},
				},
				Target: "localhost",
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			want := `{
				"containers": [{"Image": "my-app", "Status": "Up 5 minutes", "Ports": "localhost:8080"}],
				"targetHost": "localhost"
			}`
			assert.JSONEq(t, want, out.String())
		})

		t.Run("renders empty containers as empty array", func(t *testing.T) {
			toPrint := templates.PrintablePSReport{
				Containers: []templates.ContainerStatus{},
				Target:     "localhost",
			}
			var out bytes.Buffer

			err := printable.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(out.Bytes(), &got))
			assert.Equal(t, []any{}, got["containers"])
		})
	})
}
