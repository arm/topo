package e2e

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	topo := buildBinary(t)

	t.Run("writes parameters to env without changing compose yaml", func(t *testing.T) {
		projectDir := t.TempDir()
		original := configurableCompose("GREETING_NAME")
		composePath := testutil.RequireWriteComposeFile(t, projectDir, original)

		cmd := exec.Command(topo, "configure", "GREETING_NAME=World")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
		assert.Equal(t, original, testutil.RequireReadFile(t, composePath))
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{"GREETING_NAME": "World"})
	})

	t.Run("uses compose yml when compose yaml is absent", func(t *testing.T) {
		projectDir := t.TempDir()
		composePath := filepath.Join(projectDir, "compose.yml")
		original := configurableCompose("GREETING_NAME")
		testutil.RequireWriteFile(t, composePath, original)

		cmd := exec.Command(topo, "configure", "GREETING_NAME=Yml")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
		assert.Equal(t, original, testutil.RequireReadFile(t, composePath))
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{"GREETING_NAME": "Yml"})
	})

	t.Run("respects compose file flag", func(t *testing.T) {
		projectDir := t.TempDir()
		customComposePath := filepath.Join(projectDir, "custom-compose.yaml")
		originalCompose := configurableCompose("GREETING_NAME")
		customCompose := configurableCompose("CUSTOM_NAME")
		composePath := testutil.RequireWriteComposeFile(t, projectDir, originalCompose)
		testutil.RequireWriteFile(t, customComposePath, customCompose)

		cmd := exec.Command(topo, "configure", "-f", "custom-compose.yaml", "CUSTOM_NAME=Custom")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
		assert.Equal(t, customCompose, testutil.RequireReadFile(t, customComposePath))
		assert.Equal(t, originalCompose, testutil.RequireReadFile(t, composePath))
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{"CUSTOM_NAME": "Custom"})
	})

	t.Run("rejects undeclared parameters without changing compose or env files", func(t *testing.T) {
		projectDir := t.TempDir()
		original := configurableCompose("GREETING_NAME")
		composePath := testutil.RequireWriteComposeFile(t, projectDir, original)
		envPath := filepath.Join(projectDir, env.DefaultFilename)
		originalEnv := "GREETING_NAME=Original\n"
		testutil.RequireWriteFile(t, envPath, originalEnv)

		cmd := exec.Command(topo, "configure", "UNKNOWN=value")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.Error(t, err)
		assert.Contains(t, string(out), "unknown parameter: UNKNOWN")
		got := testutil.RequireReadFile(t, composePath)
		assert.Equal(t, original, got)
		assert.Equal(t, originalEnv, testutil.RequireReadFile(t, envPath))
	})
}

func configurableCompose(parameterName string) string {
	return `services:
  app:
    build:
      context: .
      args:
        GREETING_NAME: ${` + parameterName + `:-Original}

x-topo:
  name: Welcome
  parameters:
    ` + parameterName + `:
      description: Name to greet
      required: true
`
}
