package views_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/arm/topo/cli/internal/deploy"
	"github.com/arm/topo/cli/internal/output/term"
	"github.com/arm/topo/cli/internal/output/views"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerList(t *testing.T) {
	t.Run("PlainFormat", func(t *testing.T) {
		t.Run("renders container id, names, image, status, processing domain, and address", func(t *testing.T) {
			toPrint := views.ContainerList{
				Containers: []deploy.Container{
					{
						Id:               "abcdef123456",
						Names:            "project-web-1",
						Image:            "my-app",
						State:            "running",
						Status:           "Up 5 minutes",
						ProcessingDomain: "m0",
						Address:          "localhost:8080",
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "abcdef123456")
			assert.Contains(t, out.String(), "project-web-1")
			assert.Contains(t, out.String(), "my-app")
			assert.Contains(t, out.String(), "Up 5 minutes")
			assert.Contains(t, out.String(), "m0")
			assert.Contains(t, out.String(), "localhost:8080")
			assert.Contains(t, out.String(), "Container ID")
			assert.Contains(t, out.String(), "Names")
			assert.NotContains(t, out.String(), "State")
			assert.NotContains(t, out.String(), "running")
			assert.Contains(t, out.String(), "Processing Domain")
		})

		t.Run("renders multiple containers", func(t *testing.T) {
			toPrint := views.ContainerList{
				Containers: []deploy.Container{
					{Image: "web", ProcessingDomain: "Linux Host"},
					{Image: "db", ProcessingDomain: "Linux Host"},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Contains(t, out.String(), "web")
			assert.Contains(t, out.String(), "db")
			assert.Contains(t, out.String(), "Linux Host")
		})

		t.Run("renders only the header when no containers", func(t *testing.T) {
			toPrint := views.ContainerList{Containers: nil}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.Plain)

			require.NoError(t, err)
			assert.Equal(t, "Container ID   Names   Image   Status   Processing Domain   Address", out.String())
		})
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Run("renders report as valid JSON with expected fields", func(t *testing.T) {
			toPrint := views.ContainerList{
				Containers: []deploy.Container{
					{
						Id:               "abcdef123456",
						Names:            "project-web-1",
						Image:            "my-app",
						State:            "running",
						Status:           "Up 5 minutes",
						ProcessingDomain: "m0",
						Address:          "localhost:8080",
					},
				},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			want := `{
				"containers": [{"id": "abcdef123456", "names": "project-web-1", "image": "my-app", "state": "running", "status": "Up 5 minutes", "processingDomain": "m0", "address": "localhost:8080"}]
			}`
			assert.JSONEq(t, want, out.String())
		})

		t.Run("renders empty containers as empty array", func(t *testing.T) {
			toPrint := views.ContainerList{
				Containers: []deploy.Container{},
			}
			var out bytes.Buffer

			err := views.Print(toPrint, &out, term.JSON)

			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(out.Bytes(), &got))
			assert.Equal(t, []any{}, got["containers"])
		})
	})
}
