package env_test

import (
	"fmt"
	"testing"

	"github.com/arm/topo/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTargetEnv(t *testing.T) {
	t.Run("sets plain localhost without querying SSH config", func(t *testing.T) {
		t.Setenv("PATH", "")

		vars, err := env.ResolveTargetEnv("localhost")

		require.NoError(t, err)
		assert.ElementsMatch(t, vars, []string{
			fmt.Sprintf("%s=%s", env.TargetVariable, "ssh://localhost"),
			fmt.Sprintf("%s=%s", env.TargetHostnameVariable, "localhost"),
		})
	})

	t.Run("sets the resolved target environment", func(t *testing.T) {
		vars, err := env.ResolveTargetEnv("user@target.example")

		require.NoError(t, err)
		assert.ElementsMatch(t, vars, []string{
			fmt.Sprintf("%s=%s", env.TargetVariable, "ssh://user@target.example"),
			fmt.Sprintf("%s=%s", env.TargetHostnameVariable, "target.example"),
		})
	})
}
