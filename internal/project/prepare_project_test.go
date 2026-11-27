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

func TestPrepareProject(t *testing.T) {
	t.Run("sets target env vars", func(t *testing.T) {
		target := "ssh://user@hostname:8080"
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		preparedProject, err := project.PrepareProject(path, target, nil)

		require.NoError(t, err)
		require.ElementsMatch(t, preparedProject.Env, []string{
			fmt.Sprintf("%s=%s", env.TargetHostnameVariable, "hostname"),
			fmt.Sprintf("%s=%s", env.TargetVariable, target),
		})
	})

	t.Run("accepts arbitrary env files", func(t *testing.T) {
		envFiles := []string{"custom.env"}
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "services: {}\n")

		preparedProject, err := project.PrepareProject(path, "ssh://user@hostname:8080", envFiles)

		require.NoError(t, err)
		require.ElementsMatch(t, preparedProject.EnvFiles, envFiles)
	})

	t.Run("reports missing files", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.yaml")

		got, err := project.PrepareProject(path, "", nil)

		require.ErrorIs(t, err, os.ErrNotExist)
		require.Equal(t, project.PreparedProject{}, got)
	})

	t.Run("reports invalid YAML", func(t *testing.T) {
		path := testutil.RequireWriteComposeFile(t, t.TempDir(), "x-topo: [")

		got, err := project.PrepareProject(path, "", nil)

		require.Error(t, err)
		require.Equal(t, project.PreparedProject{}, got)
	})
}
