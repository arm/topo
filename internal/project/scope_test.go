package project_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/project"
	"github.com/stretchr/testify/require"
)

func TestBuildScope(t *testing.T) {
	t.Run("preserves compose file order", func(t *testing.T) {
		files := []string{"base.yaml", "override.yaml"}

		scope, err := project.BuildScope(files, "ssh://user@hostname:8080", nil)

		require.NoError(t, err)
		require.Equal(t, files, scope.ComposeFiles)
	})

	t.Run("sets target env vars", func(t *testing.T) {
		target := "ssh://user@hostname:8080"

		scope, err := project.BuildScope([]string{"compose.yaml"}, target, nil)

		require.NoError(t, err)
		require.ElementsMatch(t, scope.Env, []string{
			fmt.Sprintf("%s=%s", env.TargetHostnameVariable, "hostname"),
			fmt.Sprintf("%s=%s", env.TargetVariable, target),
		})
	})

	t.Run("accepts arbitrary env files", func(t *testing.T) {
		envFiles := []string{"custom.env"}

		scope, err := project.BuildScope([]string{"compose.yaml"}, "ssh://user@hostname:8080", envFiles)

		require.NoError(t, err)
		require.ElementsMatch(t, scope.EnvFiles, envFiles)
	})
}
