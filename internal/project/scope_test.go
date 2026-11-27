package project_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/project"
	"github.com/arm/topo/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuildScope(t *testing.T) {
	t.Run("sets target env vars", func(t *testing.T) {
		target := "ssh://user@hostname:8080"
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		scope, err := project.BuildScope(path, target, nil)

		require.NoError(t, err)
		require.ElementsMatch(t, scope.Env, []string{
			fmt.Sprintf("%s=%s", env.TargetHostnameVariable, "hostname"),
			fmt.Sprintf("%s=%s", env.TargetVariable, target),
		})
	})

	t.Run("accepts arbitrary env files", func(t *testing.T) {
		envFiles := []string{"custom.env"}
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		scope, err := project.BuildScope(path, "ssh://user@hostname:8080", envFiles)

		require.NoError(t, err)
		require.ElementsMatch(t, scope.EnvFiles, envFiles)
	})

	t.Run("reports missing files", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		got, err := project.BuildScope(path, "", nil)

		require.ErrorIs(t, err, os.ErrNotExist)
		require.Equal(t, project.Scope{}, got)
	})

	t.Run("collects environment files in include path list order", func(t *testing.T) {
		baseDir, overrideDir := t.TempDir(), t.TempDir()
		base := testutil.RequireWriteComposeFile(t, baseDir, "services:\n  app:\n    image: alpine\n")
		override := testutil.RequireWriteComposeFile(t, overrideDir, "services:\n  app:\n    command: [echo, override]\n")
		baseEnv := filepath.Join(baseDir, ".env")
		overrideEnv := filepath.Join(overrideDir, ".env.topo")
		testutil.RequireWriteFile(t, baseEnv, "GREETING=hello\n")
		testutil.RequireWriteFile(t, overrideEnv, "GREETING=override\n")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), fmt.Sprintf("include:\n  - path: [%q, %q]\n", base, override))

		got, err := project.BuildScope(path, "", nil)

		require.NoError(t, err)
		require.Equal(t, []string{baseEnv, overrideEnv}, got.EnvFiles)
	})

	t.Run("reports invalid YAML", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo: [")

		got, err := project.BuildScope(path, "", nil)

		require.Error(t, err)
		require.Equal(t, project.Scope{}, got)
	})

	t.Run("resolves nested includes in an override relative to the first file in the path list", func(t *testing.T) {
		baseDir, overrideDir := t.TempDir(), t.TempDir()
		base := testutil.RequireWriteComposeFile(t, baseDir, "services:\n  app:\n    image: alpine\n")
		override := testutil.RequireWriteComposeFile(t, overrideDir, "include: [./child/compose.yaml]\n")
		childDir := filepath.Join(baseDir, "child")
		require.NoError(t, os.Mkdir(childDir, 0o755))
		testutil.RequireWriteComposeFile(t, childDir, "services:\n  worker:\n    image: alpine\n")
		childEnv := filepath.Join(childDir, ".env")
		testutil.RequireWriteFile(t, childEnv, "GREETING=hello\n")
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), fmt.Sprintf("include:\n  - path: [%q, %q]\n", base, override))

		got, err := project.BuildScope(path, "", nil)

		require.NoError(t, err)
		require.Equal(t, []string{childEnv}, got.EnvFiles)
	})

	t.Run("accepts a shared include and collects its environment files once", func(t *testing.T) {
		root := t.TempDir()
		sharedDir := filepath.Join(root, "shared")
		require.NoError(t, os.Mkdir(sharedDir, 0o755))
		testutil.RequireWriteComposeFile(t, sharedDir, "services: {}\n")
		sharedEnv := filepath.Join(sharedDir, ".env")
		testutil.RequireWriteFile(t, sharedEnv, "GREETING=hello\n")
		testutil.RequireWriteFile(t, filepath.Join(root, "first.yaml"), "include: [./shared/compose.yaml]\n")
		testutil.RequireWriteFile(t, filepath.Join(root, "second.yaml"), "include: [./shared/compose.yaml]\n")
		path := testutil.RequireWriteComposeFile(t, root, "include: [./first.yaml, ./second.yaml]\n")

		got, err := project.BuildScope(path, "", nil)

		require.NoError(t, err)
		require.Equal(t, []string{sharedEnv}, got.EnvFiles)
	})

	t.Run("rejects an include cycle between two files", func(t *testing.T) {
		root := t.TempDir()
		path := testutil.RequireWriteComposeFile(t, root, "include: [./child.yaml]\n")
		testutil.RequireWriteFile(t, filepath.Join(root, "child.yaml"), "include: [./compose.yaml]\n")

		got, err := project.BuildScope(path, "", nil)

		require.ErrorContains(t, err, "include cycle")
		require.Equal(t, project.Scope{}, got)
	})
}
