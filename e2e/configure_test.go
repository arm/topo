package e2e

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	topo := buildBinary(t)

	t.Run("updates compose yaml parameters", func(t *testing.T) {
		projectDir := t.TempDir()
		composePath := filepath.Join(projectDir, "compose.yaml")
		testutil.RequireWriteFile(t, composePath, configurableCompose("Original"))

		cmd := exec.Command(topo, "configure", "GREETING_NAME=World")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
		want := configurableCompose("World")
		got := testutil.RequireReadFile(t, composePath)
		assert.YAMLEq(t, want, got)
	})

	t.Run("uses compose yml when compose yaml is absent", func(t *testing.T) {
		projectDir := t.TempDir()
		composePath := filepath.Join(projectDir, "compose.yml")
		testutil.RequireWriteFile(t, composePath, configurableCompose("Original"))

		cmd := exec.Command(topo, "configure", "GREETING_NAME=Yml")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.NoErrorf(t, err, "configure failed: %s", out)
		assert.Empty(t, string(out))
		want := configurableCompose("Yml")
		got := testutil.RequireReadFile(t, composePath)
		assert.YAMLEq(t, want, got)
	})

	t.Run("rejects undeclared parameters without changing the compose file", func(t *testing.T) {
		projectDir := t.TempDir()
		composePath := filepath.Join(projectDir, "compose.yaml")
		original := configurableCompose("Original")
		testutil.RequireWriteFile(t, composePath, original)

		cmd := exec.Command(topo, "configure", "UNKNOWN=value")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()

		require.Error(t, err)
		assert.Contains(t, string(out), "unknown argument: UNKNOWN")
		got := testutil.RequireReadFile(t, composePath)
		assert.Equal(t, original, got)
	})
}

func configurableCompose(greetingName string) string {
	return `services:
  app:
    build:
      context: .
      args:
        GREETING_NAME: ` + greetingName + `

x-topo:
  name: Welcome
  args:
    GREETING_NAME:
      description: Name to greet
      required: true
`
}
