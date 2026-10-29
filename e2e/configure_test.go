package e2e

import (
	"os"
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

	t.Run("reads default env files beside the first compose file", func(t *testing.T) {
		t.Setenv("TOPO_TEST_REQUIRED_VERSION", "")
		require.NoError(t, os.Unsetenv("TOPO_TEST_REQUIRED_VERSION"))
		baseDir := t.TempDir()
		overrideDir := t.TempDir()
		workingDir := t.TempDir()
		base := testutil.RequireWriteComposeFile(t, baseDir, `services:
  app:
    image: app:${TOPO_TEST_REQUIRED_VERSION:?required}
x-topo:
  parameters:
    TOPO_TEST_REQUIRED_VERSION: {}
`)
		override := testutil.RequireWriteComposeFile(t, overrideDir, "services: {}\n")
		testutil.RequireWriteFile(t, filepath.Join(baseDir, env.DefaultFilename), "TOPO_TEST_REQUIRED_VERSION=v1\n")
		cmd := exec.Command(topo, "configure", "--no-prompt", "-f", base, "-f", override)
		cmd.Dir = workingDir

		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
	})

	t.Run("rejects multi-file migration without modifying source files", func(t *testing.T) {
		projectDir := t.TempDir()
		base := filepath.Join(projectDir, "base.yaml")
		override := filepath.Join(projectDir, "override.yaml")
		baseContent := configurableCompose("GREETING_NAME")
		overrideContent := "services: {}\n"
		testutil.RequireWriteFile(t, base, baseContent)
		testutil.RequireWriteFile(t, override, overrideContent)
		cmd := exec.Command(topo, "configure", "--migrate-to-env", "-f", base, "-f", override)
		cmd.Dir = projectDir

		out, err := cmd.CombinedOutput()

		require.Error(t, err)
		assert.Contains(t, string(out), "--migrate-to-env requires exactly one Compose file")
		assert.Equal(t, baseContent, testutil.RequireReadFile(t, base))
		assert.Equal(t, overrideContent, testutil.RequireReadFile(t, override))
		assert.NoFileExists(t, filepath.Join(projectDir, env.DefaultFilename))
	})

	t.Run("writes the default env file beside the first compose file", func(t *testing.T) {
		baseDir := t.TempDir()
		overrideDir := t.TempDir()
		workingDir := t.TempDir()
		base := testutil.RequireWriteComposeFile(t, baseDir, configurableCompose("GREETING_NAME"))
		override := testutil.RequireWriteComposeFile(t, overrideDir, "services: {}\n")
		cmd := exec.Command(topo, "configure", "--no-prompt", "-f", base, "-f", override, "GREETING_NAME=World")
		cmd.Dir = workingDir

		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		testutil.RequireEnvFileValues(t, filepath.Join(baseDir, env.DefaultFilename), map[string]string{"GREETING_NAME": "World"})
		assert.NoFileExists(t, filepath.Join(overrideDir, env.DefaultFilename))
		assert.NoFileExists(t, filepath.Join(workingDir, env.DefaultFilename))
	})

	t.Run("accepts an override that replaces a legacy build argument", func(t *testing.T) {
		projectDir := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(projectDir, "base.yaml"), `services:
  app:
    build:
      context: .
      args:
        GREETING_NAME: Original
x-topo:
  parameters:
    GREETING_NAME: {}
`)
		testutil.RequireWriteFile(t, filepath.Join(projectDir, "override.yaml"), configurableCompose("GREETING_NAME"))
		cmd := exec.Command(topo, "configure", "--no-prompt", "-f", "base.yaml", "-f", "override.yaml", "GREETING_NAME=World")
		cmd.Dir = projectDir

		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{"GREETING_NAME": "World"})
	})

	t.Run("configures parameters from repeated compose file flags", func(t *testing.T) {
		projectDir := t.TempDir()
		testutil.RequireWriteFile(t, filepath.Join(projectDir, "base.yaml"), configurableCompose("BASE_NAME"))
		testutil.RequireWriteFile(t, filepath.Join(projectDir, "override.yaml"), configurableCompose("OVERRIDE_NAME"))
		cmd := exec.Command(topo, "configure", "--no-prompt", "-f", "base.yaml", "-f", "override.yaml", "BASE_NAME=Base", "OVERRIDE_NAME=Override")
		cmd.Dir = projectDir

		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{
			"BASE_NAME": "Base", "OVERRIDE_NAME": "Override",
		})
	})

	t.Run("writes parameters to env", func(t *testing.T) {
		projectDir := t.TempDir()
		original := configurableCompose("GREETING_NAME")
		testutil.RequireWriteComposeFile(t, projectDir, original)

		cmd := exec.Command(topo, "configure", "GREETING_NAME=World")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
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
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{"GREETING_NAME": "Yml"})
	})

	t.Run("respects compose file flag", func(t *testing.T) {
		projectDir := t.TempDir()
		customComposePath := filepath.Join(projectDir, "custom-compose.yaml")
		originalCompose := configurableCompose("GREETING_NAME")
		customCompose := configurableCompose("CUSTOM_NAME")
		testutil.RequireWriteComposeFile(t, projectDir, originalCompose)
		testutil.RequireWriteFile(t, customComposePath, customCompose)

		cmd := exec.Command(topo, "configure", "-f", "custom-compose.yaml", "CUSTOM_NAME=Custom")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
		testutil.RequireEnvFileValues(t, filepath.Join(projectDir, env.DefaultFilename), map[string]string{"CUSTOM_NAME": "Custom"})
	})

	t.Run("rejects undeclared parameters without changing env files", func(t *testing.T) {
		projectDir := t.TempDir()
		original := configurableCompose("GREETING_NAME")
		testutil.RequireWriteComposeFile(t, projectDir, original)
		envPath := filepath.Join(projectDir, env.DefaultFilename)
		originalEnv := "GREETING_NAME=Original\n"
		testutil.RequireWriteFile(t, envPath, originalEnv)

		cmd := exec.Command(topo, "configure", "UNKNOWN=value")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.Error(t, err)
		assert.Contains(t, string(out), "unknown parameter: UNKNOWN")
		assert.Equal(t, originalEnv, testutil.RequireReadFile(t, envPath))
	})
}

func TestComposeFileFlags(t *testing.T) {
	topo := buildBinary(t)
	for _, command := range []string{"deploy", "ps", "stop", "configure"} {
		t.Run(command+" rejects a missing file before a valid override", func(t *testing.T) {
			projectDir := t.TempDir()
			testutil.RequireWriteComposeFile(t, projectDir, "services: {}\n")
			args := []string{command, "-f", "missing.yaml", "-f", "compose.yaml"}
			if command != "configure" {
				args = append(args, "--target", "localhost")
			}
			cmd := exec.Command(topo, args...)
			cmd.Dir = projectDir

			out, err := cmd.CombinedOutput()

			require.Error(t, err)
			assert.Contains(t, string(out), filepath.Join(projectDir, "missing.yaml"))
		})
	}
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
`
}
