package e2e

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/arm/topo/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjects(t *testing.T) {
	bin := buildBinary(t)

	t.Run("lists builtin projects", func(t *testing.T) {
		cmd := exec.Command(bin, "projects")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		output := string(out)

		assert.Contains(t, output, "Hello World")
		assert.Contains(t, output, "https://github.com")
		assert.Contains(t, output, "Features:")
	})

	t.Run("keeps templates as a compatibility alias", func(t *testing.T) {
		cmd := exec.Command(bin, "templates")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		assert.Contains(t, string(out), "Hello World")
	})

	t.Run("filtering", func(t *testing.T) {
		t.Run("correctly handles the --target flag when no target description is provided", func(t *testing.T) {
			bin := buildBinary(t)
			target := testutil.StartContainer(t, testutil.DinDContainer)

			cmd := exec.Command(bin, "projects", "--target", target.SSHDestination)
			out, err := cmd.CombinedOutput()
			output := string(out)

			require.NoError(t, err, output)
			assert.Contains(t, output, "✅ Hello World")
		})
	})

	t.Run("outputs JSON when specified", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "--output", "json")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		var projects []struct {
			Name string `json:"name"`
		}
		err = json.Unmarshal(out, &projects)
		require.NoError(t, err, string(out))
		require.NotEmpty(t, projects)

		names := make([]string, 0, len(projects))
		for _, project := range projects {
			names = append(names, project.Name)
		}
		assert.Contains(t, names, "Hello World")
	})

	t.Run("outputs errors as JSON when specified", func(t *testing.T) {
		cmd := exec.Command(bin, "projects", "--output", "json", "--target", "invalid-target")
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
