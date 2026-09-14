package project_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/arm/topo/internal/project"
	"github.com/stretchr/testify/require"
)

func TestLoadScope(t *testing.T) {
	t.Run("sets target env vars", func(t *testing.T) {
		target := "ssh://user@hostname:8080"

		scope, err := project.LoadScope("compose.yaml", target)

		require.NoError(t, err)
		require.ElementsMatch(t, scope.Env, []string{
			fmt.Sprintf("%s=%s", env.TargetHostnameVariable, "hostname"),
			fmt.Sprintf("%s=%s", env.TargetVariable, target),
		})
	})
}
