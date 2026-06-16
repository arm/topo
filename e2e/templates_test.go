package e2e

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplates(t *testing.T) {
	bin := buildBinary(t)

	t.Run("lists builtin templates", func(t *testing.T) {
		cmd := exec.Command(bin, "templates")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		output := string(out)

		assert.Contains(t, output, "Hello World")
		assert.Contains(t, output, "https://github.com")
		assert.Contains(t, output, "Features:")
	})

	t.Run("filtering", func(t *testing.T) {
		t.Run("correctly handles the --target flag when no target description is provided", func(t *testing.T) {
			bin := buildBinary(t)
			target := testutil.StartContainer(t, testutil.DinDContainer)

			cmd := exec.Command(bin, "templates", "--target", target.SSHDestination)
			out, err := cmd.CombinedOutput()
			output := string(out)

			require.NoError(t, err, output)
			assert.Contains(t, output, "✅ Hello World")
		})
	})

	t.Run("outputs JSON when specified", func(t *testing.T) {
		cmd := exec.Command(bin, "templates", "--output", "json")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		var templates []struct {
			Name string `json:"name"`
		}
		err = json.Unmarshal(out, &templates)
		require.NoError(t, err, string(out))
		require.NotEmpty(t, templates)

		names := make([]string, 0, len(templates))
		for _, template := range templates {
			names = append(names, template.Name)
		}
		assert.Contains(t, names, "Hello World")
	})

	t.Run("outputs errors as JSON when specified", func(t *testing.T) {
		cmd := exec.Command(bin, "templates", "--output", "json", "--target", "invalid-target")
		out, err := cmd.CombinedOutput()
		require.Error(t, err)

		var entry map[string]interface{}
		err = json.Unmarshal(out, &entry)
		assert.NoError(t, err)
		assert.Equal(t, "ERROR", entry["level"])
		_, ok := entry["msg"].(string)
		assert.True(t, ok, "msg field should be a string")
		assert.NotNil(t, entry["time"])
	})
}
